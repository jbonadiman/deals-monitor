package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"deals_monitor/internal"
)

type request struct {
	ChannelUsername string            `json:"channelUsername"`
	MonitoredDeals  map[string]string `json:"monitoredDeals"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write(
			[]byte(fmt.Sprintf(
				"the method %q is not allowed",
				r.Method,
			)),
		)
		return
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			panic(err)
		}
	}(r.Body)

	req := request{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(
			[]byte(fmt.Sprintf(
				"error decoding request body: %v",
				err,
			)),
		)
		return
	}

	err = internal.ParseDeals(req.MonitoredDeals, req.ChannelUsername)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(
			[]byte(fmt.Sprintf(
				"error parsing deals: %v",
				err,
			)),
		)
		return
	}
}
