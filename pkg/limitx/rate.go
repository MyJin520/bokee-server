package limitx

import (
	"context"
	"time"

	"bokee/global"
	"bokee/pkg/redisx"

	"go.uber.org/zap"
)

// rateScript 原子完成「计数 → 首次写入时设置窗口 → 判断是否超限」，避免 INCR 与 EXPIRE 分离产生的竞态
const rateScript = `
local n = redis.call('INCR', KEYS[1])
if n == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
if n > tonumber(ARGV[2]) then
  return 0
end
return 1
`

// RateLimiter 固定窗口限流器：窗口内请求达到上限后拒绝。
// 与 FailLimiter 保持一致，Redis 异常时放行（fail-open），只记录日志，不阻断业务主流程。
type RateLimiter struct {
	Scope  string        // Key 作用域（如 "feedback:create:minute"），最终 Key 形如 bokee:<scope>:rate:<id>
	Window time.Duration // 统计窗口，自窗口内首次请求起算
	Max    int64         // 窗口内允许的最大请求次数
}

// Allow 判断指定标识是否可以继续请求，true 表示放行
func (l *RateLimiter) Allow(ctx context.Context, id string) bool {
	key := redisx.BuildKey(l.Scope, "rate", id)
	n, err := redisx.EvalInt(ctx, rateScript, []string{key}, l.Window.Milliseconds(), l.Max)
	if err != nil {
		global.Log.Error("limitx 限流校验失败，已放行", zap.String("key", key), zap.Error(err))
		return true
	}
	return n == 1
}
