package limitx

import (
	"context"
	"time"

	"bokee/global"
	"bokee/pkg/redisx"

	"go.uber.org/zap"
)

// FailLimiter 失败次数限流器：窗口内失败达到上限后锁定，用于登录、重置密码等防爆破场景。
// 所有 Redis 操作失败放行（fail-open）：异常仅记日志，不阻断业务主流程。
type FailLimiter struct {
	Scope  string        // Key 作用域（如 "user:login"），最终 Key 形如 bokee:<scope>:fail:<id>
	Window time.Duration // 失败计数统计窗口
	Max    int64         // 窗口内最大失败次数，达到后锁定
	Lock   time.Duration // 锁定时长
}

// failKey 失败计数 Key：bokee:<scope>:fail:<id>
func (l *FailLimiter) failKey(id string) string {
	return redisx.BuildKey(l.Scope, "fail", id)
}

// lockKey 锁定标记 Key：bokee:<scope>:lock:<id>
func (l *FailLimiter) lockKey(id string) string {
	return redisx.BuildKey(l.Scope, "lock", id)
}

// IsLocked 检查指定标识是否已被锁定，返回剩余锁定时长与是否锁定（Redis 异常放行）
func (l *FailLimiter) IsLocked(ctx context.Context, id string) (time.Duration, bool) {
	key := l.lockKey(id)
	locked, err := redisx.Exists(ctx, key)
	if err != nil {
		global.Log.Error("limitx 查询锁定状态失败", zap.String("key", key), zap.Error(err))
		return 0, false
	}
	if !locked {
		return 0, false
	}
	ttl, err := redisx.TTL(ctx, key)
	if err != nil || ttl <= 0 {
		return l.Lock, true
	}
	return ttl, true
}

// RecordFail 记录一次失败：计数 +1 并刷新统计窗口，达到阈值后写入锁定标记（Redis 异常放行）
func (l *FailLimiter) RecordFail(ctx context.Context, id string) {
	key := l.failKey(id)

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		global.Log.Error("limitx 失败计数失败", zap.String("key", key), zap.Error(err))
		return
	}
	if err := redisx.Expire(ctx, key, l.Window); err != nil {
		global.Log.Error("limitx 刷新统计窗口失败", zap.String("key", key), zap.Error(err))
	}

	if n >= l.Max {
		if err := redisx.Set(ctx, l.lockKey(id), "1", l.Lock); err != nil {
			global.Log.Error("limitx 写入锁定标记失败", zap.String("key", key), zap.Error(err))
			return
		}
		// 锁定成功，重置失败计数
		if err := redisx.Delete(ctx, key); err != nil {
			global.Log.Error("limitx 重置失败计数失败", zap.String("key", key), zap.Error(err))
		}
	}
}

// Clear 清除失败计数与锁定标记（业务成功后调用）
func (l *FailLimiter) Clear(ctx context.Context, id string) {
	if err := redisx.Delete(ctx, l.failKey(id), l.lockKey(id)); err != nil {
		global.Log.Error("limitx 清除失败计数失败", zap.String("scope", l.Scope), zap.String("id", id), zap.Error(err))
	}
}
