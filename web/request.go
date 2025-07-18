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
	UserID     *int64  `json:"user_id,omitempty"`
	CheckCode  *string `json:"check_code,omitempty"`
	Idempotent *int64  `json:"idempotent,omitempty"`
	Data       *string `json:"data,omitempty"`
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

func PostRaw(url string, req string) (string, error) {
	ret, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(req))
	if err != nil {
		return "", err
	}
	defer ret.Body.Close()

	body, err := io.ReadAll(ret.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
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
