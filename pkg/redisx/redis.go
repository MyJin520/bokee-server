package redisx

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"bokee/global"

	"github.com/redis/go-redis/v9"
)

const (
	KeyPrefix   = "bokee"     // Key统一前缀
	ErrPrefix   = "redis: "   // 错误统一前缀
	emptyMarker = "__EMPTY__" // 空值缓存标记（防止缓存穿透）
)

// getClient 安全获取Redis客户端
func getClient() *redis.Client {
	if global.Redis == nil {
		panic("Redis 未初始化，请先调用 initRedis()")
	}
	return global.Redis
}

// minDuration 取最小时长
func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// BuildKey 生成带命名空间的 Key
func BuildKey(parts ...string) string {
	var validParts []string
	validParts = append(validParts, KeyPrefix)
	for _, p := range parts {
		if p != "" {
			validParts = append(validParts, p)
		}
	}
	return strings.Join(validParts, ":")
}

// Exists 检查给定的 Key 是否存在于 Redis 中
func Exists(ctx context.Context, key string) (bool, error) {
	n, err := getClient().Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf(ErrPrefix+"Exists failed: %w", err)
	}
	return n > 0, nil
}

// Get 从 Redis 读取指定 Key 的字符串值
func Get(ctx context.Context, key string) (string, error) {
	val, err := getClient().Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf(ErrPrefix+"Get failed: %w", err)
	}
	return val, nil
}

// Set 向 Redis 写入键值对并设置过期时间
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := getClient().Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf(ErrPrefix+"Set failed: %w", err)
	}
	return nil
}

// SetNX 仅当 Key 不存在时写入并设置过期时间，返回是否写入成功（用于限流 / 去重 / 简单分布式锁）
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	ok, err := getClient().SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return false, fmt.Errorf(ErrPrefix+"SetNX failed: %w", err)
	}
	return ok, nil
}

// Delete 从 Redis 中批量删除一个或多个 Key
func Delete(ctx context.Context, keys ...string) error {
	if err := getClient().Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf(ErrPrefix+"Del failed: %w", err)
	}
	return nil
}

// TTL 获取指定 Key 的剩余生存时间（TTL）
func TTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := getClient().TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf(ErrPrefix+"TTL failed: %w", err)
	}
	return d, nil
}

// Expire 设置指定 Key 的过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := getClient().Expire(ctx, key, expiration).Err(); err != nil {
		return fmt.Errorf(ErrPrefix+"Expire failed: %w", err)
	}
	return nil
}

// Incr 对指定 Key 的数值执行加 1 操作并返回递增后的结果
func Incr(ctx context.Context, key string) (int64, error) {
	n, err := getClient().Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf(ErrPrefix+"Incr failed: %w", err)
	}
	return n, nil
}

// GetJSON 从 Redis 获取 JSON 字符串并反序列化填入 dest 结构体/指针
func GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := getClient().Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(ErrPrefix+"GetJSON failed: %w", err)
	}
	// 命中空值标记，视为无数据
	if string(data) == emptyMarker {
		return nil
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf(ErrPrefix+"Unmarshal failed: %w", err)
	}
	return nil
}

// SetJSON 将任意 Go 结构体/变量序列化为 JSON 字符串后写入 Redis
func SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf(ErrPrefix+"Marshal failed: %w", err)
	}
	if err := getClient().Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf(ErrPrefix+"SetJSON failed: %w", err)
	}
	return nil
}

// tokenBlacklistKey 生成 Token 在 Redis 中的黑名单 Key（使用 SHA-256 哈希防超长及暴露敏感信息）
func tokenBlacklistKey(tokenString string) string {
	h := sha256.Sum256([]byte(tokenString))
	return BuildKey("token", "blacklist", fmt.Sprintf("%x", h))
}

// BlacklistToken 将指定的 Token 加入 Redis 黑名单，TTL 通常设置为该 Token 的剩余有效时长
func BlacklistToken(ctx context.Context, tokenString string, ttl time.Duration) error {
	key := tokenBlacklistKey(tokenString)
	if err := getClient().Set(ctx, key, "1", ttl).Err(); err != nil {
		return fmt.Errorf(ErrPrefix+"BlacklistToken failed: %w", err)
	}
	return nil
}

// IsTokenBlacklisted 检查指定 Token 是否存在于黑名单中
func IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	key := tokenBlacklistKey(tokenString)
	n, err := getClient().Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf(ErrPrefix+"IsTokenBlacklisted failed: %w", err)
	}
	return n > 0, nil
}
