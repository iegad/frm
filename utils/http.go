package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/gox/frm/log"
)

func HttpPostJson(url string, params []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(params))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	c := &http.Client{}
	data, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer data.Body.Close()

	if data.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("POST FAILED: %v", data.StatusCode)
	}

	body, err := io.ReadAll(data.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
