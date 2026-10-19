package initialize

import (
	"context"
	"time"

	"bokee/config"
	"bokee/global"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// initRedis 初始化 Redis 客户端连接
func initRedis(cfg *config.Config) {
	if cfg.Redis.Addr == "" {
		panic("Redis 配置异常：Addr 不能为空")
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  time.Duration(cfg.Redis.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.Redis.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Redis.WriteTimeout) * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		panic("Redis 连接失败: " + err.Error())
	}

	global.Redis = client
	global.Log.Info("Redis 连接成功",
		zap.String("地址", cfg.Redis.Addr),
		zap.Int("数据库", cfg.Redis.DB),
	)
}
