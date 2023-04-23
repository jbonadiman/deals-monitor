package internal

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"deals_monitor/internal/models"
	"deals_monitor/internal/services"
)

type DailyDealsCache interface {
	GetCache(ctx context.Context, channelName string) (map[int]struct{}, error)
	PushToCache(ctx context.Context, channelName string, ids ...string) error
}

type DealMonitorNotification interface {
	NotifyDeal(title, message, url string) error
}

var (
	upstashDB       DailyDealsCache
	telegramService string
	pushoverService DealMonitorNotification
)

func initialize(ctx context.Context) {
	if upstashDB == nil {
		upstashDB = services.NewRedisClient(
			ctx,
			strings.TrimSpace(os.Getenv("REDIS_URL")),
		)
	}

	if telegramService == "" {
		// this function was idealized to be used with https://tg.i-c-a.su/
		telegramService = strings.TrimSpace(os.Getenv("TELEGRAM_ICA_HOST"))
	}

	if pushoverService == nil {
		pushoverService = services.NewPushoverService(
			strings.TrimSpace(os.Getenv("PUSHOVER_TOKEN")),
			strings.TrimSpace(os.Getenv("PUSHOVER_USER")),
		)
	}
}

func ParseDeals(
	monitoredDeals map[string]string,
	channelUsername string,
) error {
	ctx := context.Background()

	initialize(ctx)
	var wg sync.WaitGroup
	var messages []*models.Message
	var err error
	compiledPatterns := make(map[string]*regexp.Regexp, len(monitoredDeals))

	wg.Add(1)
	go func() { // fetch telegram messages
		defer wg.Done()
		messages, err = services.GetTelegramMessages(
			telegramService,
			channelUsername,
			20,
		)
	}()

	dailyCache, err := upstashDB.GetCache(ctx, channelUsername)
	if err != nil {
		return err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for dealName, pattern := range monitoredDeals {
			compiledPatterns[dealName] = regexp.MustCompile(pattern)
		}
	}()

	wg.Wait()
	if err != nil {
		return err
	}

	cacheBatch := make([]string, 0, len(messages))

	for _, msg := range messages {
		if msg.GetDate().Day() != time.Now().Day() { // post is not from today
			continue
		}

		if _, ok := dailyCache[msg.Id]; !ok { // post not on cache
			cacheBatch = append(cacheBatch, strconv.Itoa(msg.Id))

			for dealName, pattern := range compiledPatterns {
				if pattern.MatchString(msg.Content) {

					wg.Add(1)
					go func() { // notify deal
						defer wg.Done()
						err = pushoverService.NotifyDeal(
							fmt.Sprintf("💰 new deal for %q!", dealName),
							fmt.Sprintf("found on %s", msg.Channel.Title),
							msg.GetLink(),
						)
					}()

					break // no need to check other deals
				}
			}
		}
	}

	if len(cacheBatch) > 0 {
		wg.Add(1)
		go func() { // write in batch to cache
			defer wg.Done()
			err = upstashDB.PushToCache(ctx, channelUsername, cacheBatch...)
		}()
	}

	wg.Wait()
	if err != nil {
		return err
	}

	return nil
}
