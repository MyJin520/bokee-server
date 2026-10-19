package cachex

import (
	"context"
	"fmt"

	"bokee/global"
	"bokee/pkg/redisx"

	"go.uber.org/zap"
)

// 业务缓存命名空间
const (
	NSUser    = "user"
	NSArticle = "article"
	NSRole    = "role"
)

// Key 生成业务缓存 Key
func Key(parts ...any) string {
	strs := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == nil {
			continue
		}
		strs = append(strs, fmt.Sprint(p))
	}
	return redisx.BuildKey(strs...)
}

// Invalidate 批量失效缓存 Key
func Invalidate(ctx context.Context, keys ...string) {
	if len(keys) == 0 {
		return
	}
	if err := redisx.Delete(ctx, keys...); err != nil {
		global.Log.Error("失效缓存失败", zap.Strings("keys", keys), zap.Error(err))
	}
}
