package internal

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"deals_monitor/internal/models"
	"deals_monitor/internal/services"
)

type DealsService interface {
	GetTodayDeals(ctx context.Context, channelName string) (
		[]*models.Message,
		error,
	)
}

type DealsCache interface {
	GetCache(ctx context.Context, channelName string) (
		map[string]struct{},
		error,
	)
	PushToCache(ctx context.Context, channelName string, ids ...string) error
}

type DealsMonitor interface {
	NotifyDeal(title, message, url string) error
}

var (
	upstashDB       DealsCache
	telegramService string
	pushoverService DealsMonitor
)

func initialize(ctx context.Context) {
	if upstashDB == nil {
		upstashDB = services.NewRedisClient(
			ctx,
			strings.TrimSpace(os.Getenv("REDIS_URL")),
		)
	}

	if telegramService == "" {
		telegramService = strings.TrimSpace(os.Getenv("TELEGRAM_HOST"))
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
	var channelHistory models.TelegramResponse
	compiledPatterns := make(map[string]*regexp.Regexp, len(monitoredDeals))

	dailyCache, err := upstashDB.GetCache(ctx, channelUsername)
	if err != nil {
		return err
	}

	wg.Add(1)
	go func() { // fetch channel history
		defer wg.Done()
		channelHistory, err = services.GetTelegramMessages(
			telegramService,
			channelUsername,
		)
	}()

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

	cacheBatch := make([]string, 0, len(channelHistory.Messages))

	for _, msg := range channelHistory.Messages {
		if msg.GetDate().Day() != time.Now().Day() { // message is not from today
			continue
		}

		if _, ok := dailyCache[msg.Id]; !ok { // message not on cache
			cacheBatch = append(cacheBatch, msg.Id)

			for dealName, pattern := range compiledPatterns {
				if pattern.MatchString(msg.Content) {
					if err = pushoverService.NotifyDeal(
						fmt.Sprintf("💰 new deal for %q!", dealName),
						fmt.Sprintf(
							"found on %s",
							channelHistory.Channel.Name,
						),
						channelHistory.GetMessageLink(msg.Id),
					); err != nil {
						return err
					}
					break // no need to check other deals
				}
			}
		}
	}

	if len(cacheBatch) > 0 {
		if err = upstashDB.PushToCache(
			ctx,
			channelUsername,
			cacheBatch...,
		); err != nil {
			return err
		}
	}

	return nil
}
