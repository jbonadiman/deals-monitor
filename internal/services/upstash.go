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
	Result []string `json:"result"`
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
			"%s/LRANGE/%s/0/-1?_token=%s",
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
		integer, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		cache[integer] = struct{}{}
	}

	return cache, nil
}

func (u *UpstashDB) PushToCache(channelName string, ids ...string) error {
	cacheKey := getCacheKey(channelName)

	response, err := http.Get(
		fmt.Sprintf(
			"%s/RPUSH/%s/%s?_token=%s",
			u.Host,
			cacheKey,
			strings.Join(ids, "/"),
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
