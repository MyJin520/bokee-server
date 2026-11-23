package ws

import (
	"context"
	"fmt"
	"net/http"

	"bokee/global"
	"bokee/pkg/redisx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HandleUpgrade WebSocket 握手入口
func HandleUpgrade(c *gin.Context) {
	h := defaultHub
	if h == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": 503, "msg": "实时服务未初始化"})
		return
	}

	uid, err := consumeTicket(c.Request.Context(), c.Query("ticket"))
	if err != nil {
		global.Log.Warn("WebSocket 连接凭证无效", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "msg": "连接凭证无效或已过期"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		global.Log.Warn("WebSocket 升级失败", zap.Uint("user_id", uid), zap.Error(err))
		return
	}
	h.Serve(uid, conn)
}

// consumeTicket 原子消费一次性票据（Redis GETDEL）
func consumeTicket(ctx context.Context, ticket string) (uint, error) {
	if ticket == "" {
		return 0, fmt.Errorf("ticket 不能为空")
	}
	key := redisx.BuildKey("im", "ws", "ticket", ticket)
	id, err := global.Redis.GetDel(ctx, key).Uint64()
	if err != nil || id == 0 {
		return 0, fmt.Errorf("ticket 无效或已过期")
	}
	return uint(id), nil
}
