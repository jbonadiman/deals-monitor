package services

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"

	"deals_monitor/internal/models"
)

func GetTelegramMessages(
	host string,
	channelUsername string,
	limit int,
) ([]*models.Message, error) {
	limitParsed := math.Max(
		10,
		math.Min(100, float64(limit)),
	) // limit between 10 and 100
	limit = int(limitParsed)

	response, err := http.Get(
		fmt.Sprintf(
			"%s/json/%s?limit=%d",
			host,
			channelUsername,
			limit,
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

	var telegramResponse models.TelegramResponse
	err = json.NewDecoder(response.Body).Decode(&telegramResponse)
	if err != nil {
		return nil, err
	}

	for _, msg := range telegramResponse.Messages {
		msg.Channel = &telegramResponse.Chats[0]
	}

	return telegramResponse.Messages, nil
}
