package redisx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type User struct {
	ID    int
	Name  string
	Other string
}

var (
	client      *redis.Client
	ctx         = context.Background()
	userList    []User
	cachePrefix = "user"
)

func TestRedis(t *testing.T) {
	initRedis()
	if client == nil {
		t.Fatal("redis 连接失败")
	}
	fmt.Println("redis 连接成功")

	addCache()

	cache, err := getCache(5)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			t.Log("缓存未命中")
			return
		}
		t.Fatalf("获取数据失败: %v", err)
	}
	fmt.Printf("获取到用户: %+v\n", cache)
}

func initRedis() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "123456",
		DB:       0,
	})
	client = rdb
}

func addCache() {
	user_01 := User{ID: 1, Name: "kim", Other: "唱跳rap"}
	user_02 := User{ID: 2, Name: "Tom", Other: "嘻嘻哈哈"}
	user_03 := User{ID: 3, Name: "Ani", Other: "开开心心"}

	userList = []User{user_01, user_02, user_03}
	for _, user := range userList {
		key := fmt.Sprintf("%s_%d", cachePrefix, user.ID)
		data, err := json.Marshal(user)
		if err != nil {
			panic(err)
		}
		client.Set(ctx, key, data, 5*time.Minute)
	}
}

func getCache(id int) (User, error) {
	key := fmt.Sprintf("%s_%d", cachePrefix, id)
	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return User{}, err
	}
	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return User{}, err
	}
	return user, nil
}
