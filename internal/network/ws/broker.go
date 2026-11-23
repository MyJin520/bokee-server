package ws

import (
	"context"
	"sync"
	"time"

	"bokee/global"
	"bokee/pkg/imx"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Broker 单实例只维护一个 Redis Pub/Sub 连接，按本机在线用户动态订阅
type Broker struct {
	ps   *redis.PubSub
	hub  *Hub
	mu   sync.Mutex
	refs map[uint]int
}

func NewBroker() *Broker {
	return &Broker{
		ps:   global.Redis.Subscribe(context.Background()),
		refs: make(map[uint]int),
	}
}

func (b *Broker) Run() {
	ch := b.ps.Channel(redis.WithChannelSize(1024))
	for msg := range ch {
		uid, ok := imx.ParseUserChannel(msg.Channel)
		if !ok {
			continue
		}
		b.hub.Deliver(uid, []byte(msg.Payload))
	}
}

func (b *Broker) Track(uid uint) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refs[uid]++
	if b.refs[uid] != 1 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := b.ps.Subscribe(ctx, imx.UserChannel(uid)); err != nil {
		// 订阅失败则回滚计数，后续新连接会再次尝试订阅
		global.Log.Warn("Redis 订阅私信频道失败", zap.Uint("user_id", uid), zap.Error(err))
		b.refs[uid]--
		if b.refs[uid] <= 0 {
			delete(b.refs, uid)
		}
	}
}

func (b *Broker) Untrack(uid uint) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.refs[uid] > 1 {
		b.refs[uid]--
		return
	}
	delete(b.refs, uid)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := b.ps.Unsubscribe(ctx, imx.UserChannel(uid)); err != nil {
		global.Log.Warn("Redis 退订私信频道失败", zap.Uint("user_id", uid), zap.Error(err))
	}
}

func (b *Broker) Close() {
	_ = b.ps.Close()
}
