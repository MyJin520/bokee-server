package ws

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"bokee/config"
	"bokee/global"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const shardCount = 64

type shard struct {
	mu    sync.RWMutex
	conns map[uint]map[*client]struct{}
}

type Hub struct {
	shards         [shardCount]shard
	broker         *Broker
	upgrader       websocket.Upgrader
	maxPerUser     int
	sendBuffer     int
	pingInterval   time.Duration
	originPatterns []string
}

var (
	defaultHub *Hub
	hubOnce    sync.Once
)

// Init 启动私信实时模块（进程内只执行一次）
func Init(conf config.IMConfig) {
	hubOnce.Do(func() {
		h := &Hub{
			maxPerUser:     conf.WsMaxConnPerUser,
			sendBuffer:     conf.WsSendBuffer,
			pingInterval:   time.Duration(conf.WsPingInterval) * time.Second,
			originPatterns: conf.WsOriginPatterns,
		}
		for i := range h.shards {
			h.shards[i].conns = make(map[uint]map[*client]struct{})
		}
		h.upgrader = websocket.Upgrader{
			ReadBufferSize:   1024,
			WriteBufferSize:  4096,
			HandshakeTimeout: 10 * time.Second,
			CheckOrigin:      h.checkOrigin,
		}

		h.broker = NewBroker()
		h.broker.hub = h
		defaultHub = h
		go h.broker.Run()

		global.Log.Info("私信实时模块已启动",
			zap.Int("shards", shardCount),
			zap.Int("max_conn_per_user", h.maxPerUser))
	})
}

// Shutdown 关闭所有连接（如后续接入优雅退出可调用）
func Shutdown() {
	h := defaultHub
	if h == nil {
		return
	}
	h.broker.Close()
	for i := range h.shards {
		sh := &h.shards[i]
		sh.mu.RLock()
		clients := make([]*client, 0)
		for _, set := range sh.conns {
			for c := range set {
				clients = append(clients, c)
			}
		}
		sh.mu.RUnlock()
		for _, c := range clients {
			c.close()
		}
	}
}

// Serve 接管一条已认证的长连接
func (h *Hub) Serve(uid uint, conn *websocket.Conn) {
	c := newClient(uid, conn, h.sendBuffer)
	if !h.register(c) {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "连接数超出限制"),
			time.Now().Add(2*time.Second),
		)
		_ = conn.Close()
		return
	}

	h.broker.Track(uid)
	defer func() {
		h.unregister(c)
		h.broker.Untrack(uid)
		c.close()
	}()

	go c.writeLoop(h.pingInterval)
	c.readLoop(h.pingInterval)
}

// Deliver 向本机某用户的所有连接投递事件
func (h *Hub) Deliver(uid uint, payload []byte) {
	sh := h.shardFor(uid)
	sh.mu.RLock()
	set := sh.conns[uid]
	slow := make([]*client, 0)
	for c := range set {
		select {
		case c.send <- payload:
		default:
			slow = append(slow, c)
		}
	}
	sh.mu.RUnlock()

	for _, c := range slow {
		go c.close()
	}
}

func (h *Hub) shardFor(uid uint) *shard {
	return &h.shards[uid%shardCount]
}

func (h *Hub) register(c *client) bool {
	sh := h.shardFor(c.uid)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	set := sh.conns[c.uid]
	if set == nil {
		set = make(map[*client]struct{})
		sh.conns[c.uid] = set
	}
	if h.maxPerUser > 0 && len(set) >= h.maxPerUser {
		return false
	}
	set[c] = struct{}{}
	return true
}

func (h *Hub) unregister(c *client) {
	sh := h.shardFor(c.uid)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	set := sh.conns[c.uid]
	if set == nil {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(sh.conns, c.uid)
	}
}

// checkOrigin 校验浏览器 Origin：同源放行；本地开发（前后端端口不同）时同为回环地址也放行；其余匹配配置
func (h *Hub) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	// 开发环境常见形态：页面在 localhost:5173/5174，Vite 代理后 Host 变成 localhost:8888，
	// 两端都是本机回环地址时视为可信来源，避免每换一个前端端口就要改配置
	if isLoopbackHost(u.Hostname()) && isLoopbackHost(hostnameOf(r.Host)) {
		return true
	}
	for _, pattern := range h.originPatterns {
		p := strings.TrimSpace(pattern)
		if p == "" {
			continue
		}
		if p == "*" {
			return true
		}
		if strings.EqualFold(p, u.Host) {
			return true
		}
		if strings.HasPrefix(p, "*.") {
			suffix := strings.ToLower(strings.TrimPrefix(p, "*"))
			if strings.HasSuffix(strings.ToLower(u.Hostname()), suffix) {
				return true
			}
		}
	}
	global.Log.Warn("WebSocket Origin 被拒绝",
		zap.String("origin", origin),
		zap.String("host", r.Host),
		zap.Strings("patterns", h.originPatterns))
	return false
}

// isLoopbackHost 判断主机名是否为本机回环地址
func isLoopbackHost(host string) bool {
	h := strings.ToLower(strings.Trim(host, "[]"))
	if h == "" {
		return false
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback()
	}
	return h == "localhost"
}

// hostnameOf 去掉 host:port 里的端口
func hostnameOf(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}
