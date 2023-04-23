package models

import (
	"fmt"
	"strconv"
	"time"
)

type TelegramResponse struct {
	Messages []*Message `json:"messages"`
	Chats    []Channel  `json:"chats"`
}

type Channel struct {
	Title    string `json:"title"`
	Username string `json:"username"`
}

type Message struct {
	Id        int    `json:"id"`
	DateEpoch int64  `json:"date"`
	Content   string `json:"message"`
	Channel   *Channel
}

func (m Message) GetDate() time.Time {
	return time.Unix(m.DateEpoch, 0).UTC()
}

func (m Message) GetLink() string {
	return fmt.Sprintf(
		"https://t.me/%s/%s",
		m.Channel.Username,
		strconv.Itoa(m.Id),
	)
}
