package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type client struct {
	uid       uint
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func newClient(uid uint, conn *websocket.Conn, buffer int) *client {
	if buffer <= 0 {
		buffer = 128
	}
	return &client{
		uid:  uid,
		conn: conn,
		send: make(chan []byte, buffer),
		done: make(chan struct{}),
	}
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (c *client) writeLoop(pingInterval time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				c.close()
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close()
				return
			}
		}
	}
}

// readLoop 只负责维持连接与处理 pong；业务消息统一走 HTTP 发送
func (c *client) readLoop(pingInterval time.Duration) {
	pongWait := pingInterval * 2
	if pongWait < 60*time.Second {
		pongWait = 60 * time.Second
	}

	c.conn.SetReadLimit(1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}
