package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const KeyFormat = "deals_monitor:%s:%s"

type UpstashResponse struct {
	Result []int `json:"result"`
}

type UpstashDB struct {
	Host  string
	token string
}

func NewUpstashDB(host string, token string) *UpstashDB {
	return &UpstashDB{
		Host:  host,
		token: token,
	}
}

func (u *UpstashDB) GetCache(channelName string) (map[int]struct{}, error) {
	cacheKey := getCacheKey(channelName)

	response, err := http.Get(
		fmt.Sprintf(
			"%s/GET/%s?_token=%s",
			u.Host,
			cacheKey,
			u.token,
		),
	)
	if err != nil {
		return nil, err
	}

	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(response.Body)

	if response.StatusCode > 300 {
		return nil, fmt.Errorf("error getting key %q", cacheKey)
	}

	var upstashResponse UpstashResponse
	err = json.NewDecoder(response.Body).Decode(&upstashResponse)
	if err != nil {
		return nil, err
	}

	var cache = make(map[int]struct{})
	for _, id := range upstashResponse.Result {
		cache[id] = struct{}{}
	}

	return cache, nil
}

func (u *UpstashDB) CreateCache(channelName string) error {
	cacheKey := getCacheKey(channelName)

	response, err := http.Get(
		fmt.Sprintf(
			"%s/SET/%s/%s/EX/%d?_token=%s",
			u.Host,
			cacheKey,
			"[]",
			int(24*time.Hour/time.Second),
			u.token,
		),
	)
	if err != nil {
		return err
	}

	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(response.Body)

	if response.StatusCode > 300 {
		return fmt.Errorf(
			"error creating cache with key %q",
			cacheKey,
		)
	}

	return nil
}

func (u *UpstashDB) PushToCache(channelName string, ids ...int) error {
	cacheKey := getCacheKey(channelName)

	var builder strings.Builder
	for _, id := range ids {
		builder.WriteString(strconv.Itoa(id))
		builder.WriteString("/")
	}

	response, err := http.Get(
		fmt.Sprintf(
			"%s/RPUSH/%s/%s?_token=%s",
			u.Host,
			cacheKey,
			strings.TrimRight(builder.String(), "/"),
			u.token,
		),
	)
	if err != nil {
		return err
	}

	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(response.Body)

	if response.StatusCode > 300 {
		return fmt.Errorf(
			"error pushing to cache %q", cacheKey,
		)
	}

	return nil
}

func getCacheKey(channelName string) string {
	return fmt.Sprintf(
		KeyFormat,
		time.Now().Format("20060102"),
		channelName,
	)
}
