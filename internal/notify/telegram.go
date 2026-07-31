package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegram(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TelegramNotifier) Name() string { return "telegram" }

func (t *TelegramNotifier) SendCritical(p Payload) error {
	if t.botToken == "" || t.chatID == "" {
		return nil
	}
	text := fmt.Sprintf(
		"Pulse99 -- Critical Alert\n\n"+
			"[ Target ]     %s\n"+
			"[ URL ]        %s\n"+
			"[ Status ]     DOWN\n"+
			"[ Failures ]   %d consecutive\n"+
			"[ Cause ]      %s\n"+
			"[ Timestamp ]  %s UTC",
		p.Name, p.URL, p.Failures, p.Reason, p.Timestamp.UTC().Format(time.RFC3339),
	)
	return t.send(text)
}

func (t *TelegramNotifier) SendRecovery(p Payload) error {
	if t.botToken == "" || t.chatID == "" {
		return nil
	}
	text := fmt.Sprintf(
		"Pulse99 -- Recovery Alert\n\n"+
			"[ Target ]     %s\n"+
			"[ URL ]        %s\n"+
			"[ Status ]     ONLINE\n"+
			"[ Latency ]    %dms\n"+
			"[ Timestamp ]  %s UTC",
		p.Name, p.URL, p.LatencyMs, p.Timestamp.UTC().Format(time.RFC3339),
	)
	return t.send(text)
}

func (t *TelegramNotifier) send(text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	payload := struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}{
		ChatID: t.chatID,
		Text:   text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := t.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
