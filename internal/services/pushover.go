package services

import (
	"fmt"
	"io"
	"net/http"
)

const PushoverAPIHost = "https://api.pushover.net/1/messages.json"

type PushoverService struct {
	token string
	user  string
}

func NewPushoverService(token string, user string) *PushoverService {
	return &PushoverService{
		token: token,
		user:  user,
	}
}

func (p *PushoverService) NotifyDeal(title, message, url string) error {
	response, err := http.Post(
		fmt.Sprintf(
			"%s?token=%s&user=%s&title=%s&message=%s&url=%s",
			PushoverAPIHost,
			p.token,
			p.user,
			title,
			message,
			url,
		), "application/json", nil,
	)
	if err != nil {
		return err
	}

	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(response.Body)

	return nil
}
