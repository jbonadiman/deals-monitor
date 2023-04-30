package models

import (
	"fmt"
	"time"
)

type Channel struct {
	Name     string    `json:"name"`
	Username string    `json:"username"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Id        int    `json:"id"`
	DateEpoch int64  `json:"dateEpoch"`
	Content   string `json:"message"`
}

func (m Message) GetDate() time.Time {
	return time.Unix(m.DateEpoch, 0).UTC()
}

func (c Channel) GetMessageLink(id int) string {
	return fmt.Sprintf(
		"https://t.me/%s/%d",
		c.Username,
		id,
	)
}
