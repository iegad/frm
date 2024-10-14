package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gox/frm/log"
)

type BasicRequest struct {
	UserID     *int32      `json:"user_id,omitempty"`
	Token      *string     `json:"token,omitempty"`
	Idempotent *int64      `json:"idempotent,omitempty"`
	Data       interface{} `json:"data,omitempty"`
}

func PostJSON(url string, req, rsp any) error {
	data, _ := json.Marshal(req)
	ret, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, rsp)
}

func Post(url string, req string, rsp any) error {
	log.Debug(url)
	log.Debug(req)
	ret, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(req))
	if err != nil {
		return err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return err
	}

	log.Debug(string(body))
	return json.Unmarshal(body, rsp)
}

func Get(url string, rsp any) error {
	ret, err := http.Get(url)
	if err != nil {
		return err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, rsp)
}
