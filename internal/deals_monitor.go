package internal

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"deals_monitor/internal/models"
	"deals_monitor/internal/services"
)

type DailyDealsCache interface {
	GetCache(channelName string) (map[int]struct{}, error)
	PushToCache(channelName string, ids ...string) error
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
		// this function was idealized to be used with https://tg.i-c-a.su/
		telegramService = os.Getenv("TELEGRAM_ICA_HOST")
	}

	if pushoverService == nil {
		pushoverService = services.NewPushoverService(
			os.Getenv("PUSHOVER_TOKEN"),
			os.Getenv("PUSHOVER_USER"),
		)
	}
}

func ParseDeals(
	monitoredDeals map[string]string,
	channelUsername string,
) error {
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
			channelUsername,
			20,
		)
	}()

	dailyCache, err := upstashDB.GetCache(channelUsername)
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
			err = upstashDB.PushToCache(channelUsername, cacheBatch...)
		}()
	}

	wg.Wait()
	if err != nil {
		return err
	}

	return nil
}
