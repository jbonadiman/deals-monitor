package models

import (
	"fmt"
	"time"
)

type TelegramResponse struct {
	Channel  Channel   `json:"channel"`
	Messages []Message `json:"messages"`
}

type Channel struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	ImageURL string `json:"image"`
}

type Message struct {
	Id        string `json:"id"`
	DateEpoch int64  `json:"dateEpoch"`
	Content   string `json:"content"`
}

func (m Message) GetDate() time.Time {
	return time.Unix(m.DateEpoch, 0).UTC()
}

func (t TelegramResponse) GetMessageLink(id string) string {
	return fmt.Sprintf(
		"https://t.me/%s/%s",
		t.Channel.Username,
		id,
	)
}
