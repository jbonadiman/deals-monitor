package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const KeyFormat = "deals_monitor:%s:%s"

type RedisConfig struct {
	Host     string
	Port     int
	Password string
}

type UpstashDB struct {
	client *redis.Client
}

func NewRedisClient(redisConfig RedisConfig) *UpstashDB {
	redisUrl := fmt.Sprintf(
		"redis://:%s@%s:%d/0",
		redisConfig.Password,
		redisConfig.Host,
		redisConfig.Port,
	)

	fmt.Printf("redisUrl: %s", redisUrl)

	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		panic(err)
	}

	return &UpstashDB{
		client: redis.NewClient(opt),
	}
}

func (redis *UpstashDB) GetCache(
	ctx context.Context,
	channelName string,
) (map[int]struct{}, error) {
	cacheKey := getCacheKey(channelName)

	redisArray, err := redis.client.LRange(ctx, cacheKey, 0, -1).Result()
	if err != nil {
		return nil, err
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

func (redis *UpstashDB) PushToCache(
	ctx context.Context,
	channelName string,
	ids ...string,
) error {
	cacheKey := getCacheKey(channelName)

	var wg sync.WaitGroup
	var err error

	wg.Add(1)
	go func() {
		defer wg.Done()

		err = redis.client.RPush(ctx, cacheKey, ids).Err()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err = redis.client.ExpireNX(ctx, cacheKey, 24*time.Hour).Err()
	}()

	wg.Wait()
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
