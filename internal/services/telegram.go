package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"deals_monitor/internal/models"
)

func GetTelegramMessages(
	host string,
	channelUsername string,
) (*models.Channel, error) {
	t := time.Now()
	tMinus := t.AddDate(0, 0, -1)

	response, err := http.Get(
		fmt.Sprintf(
			"%s/api/channel/messages?channelId=%s&fromDateUTC=%d&toDateUTC=%d",
			host,
			channelUsername,
			tMinus.Unix(),
			t.Unix(),
		),
	)
	if err != nil {
		return nil, err
	}

	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(response.Body)

	if response.StatusCode > 300 {
		return nil, fmt.Errorf(
			"error getting messages from channel %q: %q",
			channelUsername,
			response.Status,
		)
	}

	var telegramResponse models.Channel
	err = json.NewDecoder(response.Body).Decode(&telegramResponse)
	if err != nil {
		return nil, err
	}

	return &telegramResponse, nil
}
