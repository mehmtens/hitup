package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

type RedisBroker struct {
	client *redis.Client
}

func NewRedisBroker(addr string) (*RedisBroker, error) {
	var opt *redis.Options
	var err error

	// Eğer redis:// veya rediss:// URL formatındaysa otomatik parse et
	if strings.HasPrefix(addr, "redis://") || strings.HasPrefix(addr, "rediss://") {
		opt, err = redis.ParseURL(addr)
		if err != nil {
			return nil, fmt.Errorf("geçersiz redis url: %w", err)
		}
	} else {
		opt = &redis.Options{
			Addr: addr,
		}
	}

	rdb := redis.NewClient(opt)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis bağlantı hatası: %w", err)
	}

	log.Println("Redis bağlantısı başarılı.")
	return &RedisBroker{client: rdb}, nil
}

func (b *RedisBroker) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, channel, data).Err()
}

func (b *RedisBroker) SubscribeAll(ctx context.Context, onMessage func(channel string, payload []byte)) {
	pubsub := b.client.PSubscribe(ctx, "chat:*")
	ch := pubsub.Channel()

	go func() {
		defer pubsub.Close()
		for msg := range ch {
			onMessage(msg.Channel, []byte(msg.Payload))
		}
	}()
}

func (b *RedisBroker) SetUserOnline(ctx context.Context, userID int) error {
	return b.client.SAdd(ctx, "online_users", userID).Err()
}

func (b *RedisBroker) SetUserOffline(ctx context.Context, userID int) error {
	return b.client.SRem(ctx, "online_users", userID).Err()
}

func (b *RedisBroker) GetOnlineUserIDs(ctx context.Context) map[int]bool {
	members, err := b.client.SMembers(ctx, "online_users").Result()
	res := make(map[int]bool)
	if err != nil {
		return res
	}
	for _, m := range members {
		if id, err := strconv.Atoi(m); err == nil {
			res[id] = true
		}
	}
	return res
}
