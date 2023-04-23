package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const KeyFormat = "deals_monitor:%s:%s"

type UpstashDB struct {
	client *redis.Client
}

func NewRedisClient(ctx context.Context, url string) *UpstashDB {
	opt, err := redis.ParseURL(url)
	if err != nil {
		panic(err)
	}

	opt.ConnMaxIdleTime = 5 * time.Minute
	client := redis.NewClient(opt)

	err = client.Ping(ctx).Err()
	if err != nil {
		panic(err)
	}

	return &UpstashDB{
		client: client,
	}
}

func (r *UpstashDB) GetCache(
	ctx context.Context,
	channelName string,
) (map[int]struct{}, error) {
	cacheKey := getCacheKey(channelName)

	redisArray := r.client.LRange(ctx, cacheKey, 0, -1).Val()
	if redisArray == nil {
		return nil, nil
	}

	var cache = make(map[int]struct{})
	for _, id := range redisArray {
		integer, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		cache[integer] = struct{}{}
	}

	return cache, nil
}

func (r *UpstashDB) PushToCache(
	ctx context.Context,
	channelName string,
	ids ...string,
) error {
	cacheKey := getCacheKey(channelName)

	pipe := r.client.Pipeline()

	pipe.RPush(ctx, cacheKey, ids)
	pipe.Expire(ctx, cacheKey, 24*time.Hour)

	cmd, err := pipe.Exec(ctx)

	log.Println(cmd)
	if err != nil {
		return err
	}

	return nil
}

func getCacheKey(channelName string) string {
	return fmt.Sprintf(
		KeyFormat,
		time.Now().Format("20060102"),
		channelName,
	)
}
