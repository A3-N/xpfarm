package telegram

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	Token  string
	ChatID string
}

func NewClient(token, chatID string) *Client {
	return &Client{
		Token:  token,
		ChatID: chatID,
	}
}

func (c *Client) SendNotification(message string) error {
	if c.Token == "" || c.ChatID == "" {
		return fmt.Errorf("telegram token or chat_id not set")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.Token)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.PostForm(apiURL, url.Values{
		"chat_id":    {c.ChatID},
		"text":       {message},
		"parse_mode": {"Markdown"},
	})
	if err != nil {
		log.Printf("[Telegram] Error sending message: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Telegram] Unexpected status code: %d", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
