package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gox/frm/log"
	"github.com/gox/frm/utils"
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

	body, _ := io.ReadAll(ret.Body)
	log.Debug(*utils.Bytes2Str(body))
	err = json.Unmarshal(body, rsp)
	if err != nil {
		return err
	}

	return nil
}
