package tg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func Send(token, chatID, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	body := map[string]string{
		"chat_id": chatID,
		"text":    message,
	}

	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message: %s", resp.Status)
	}

	return nil
}
