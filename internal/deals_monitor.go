package internal

import (
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"deals_monitor/internal/models"
	"deals_monitor/internal/services"
)

type DailyDealsCache interface {
	GetCache(channelName string) (map[int]struct{}, error)
	CreateCache(channelName string) error
	PushToCache(channelName string, ids ...int) error
}

type DealMonitorNotification interface {
	NotifyDeal(title, message, url string) error
}

var (
	upstashDB       DailyDealsCache
	telegramService string
	pushoverService DealMonitorNotification
)

func initialize() {
	if upstashDB == nil {
		upstashDB = services.NewUpstashDB(
			os.Getenv("UPSTASH_HOST"),
			os.Getenv("UPSTASH_TOKEN"),
		)
	}

	if telegramService == "" {
		telegramService = os.Getenv("TELEGRAM_ICA_HOST")
	}

	if pushoverService == nil {
		pushoverService = services.NewPushoverService(
			os.Getenv("PUSHOVER_TOKEN"),
			os.Getenv("PUSHOVER_USER"),
		)
	}
}

func ParseDeals(monitoredDeals map[string]string, channelName string) error {
	initialize()
	var wg sync.WaitGroup
	var messages []models.Message
	var err error
	compiledPatterns := make(map[string]*regexp.Regexp, len(monitoredDeals))

	wg.Add(1)
	go func() { // fetch telegram messages
		defer wg.Done()
		messages, err = services.GetTelegramMessages(
			telegramService,
			channelName,
			20,
		)
	}()

	dailyCache, err := upstashDB.GetCache(channelName)
	if err != nil {
		return err
	}

	if dailyCache == nil || len(dailyCache) == 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = upstashDB.CreateCache(channelName)
		}()
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

	for _, msg := range messages {
		if msg.GetDate().Day() != time.Now().Day() { // post is not from today
			continue
		}

		if _, ok := dailyCache[msg.Id]; !ok { // post not on cache
			wg.Add(1)
			go func() { // add post to cache
				defer wg.Done()
				err = upstashDB.PushToCache(channelName, msg.Id)
			}()

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
				}
			}

			wg.Wait()
		}
	}
	if err != nil {
		return err
	}

	return nil
}
