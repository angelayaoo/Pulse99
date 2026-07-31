package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewDiscord(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *DiscordNotifier) Name() string { return "discord" }

func (d *DiscordNotifier) SendCritical(p Payload) error {
	if d.webhookURL == "" {
		return nil
	}
	embed := map[string]interface{}{
		"title":       fmt.Sprintf("Critical: %s is DOWN", p.Name),
		"description": fmt.Sprintf("Target **%s** has been **offline** for **%d** consecutive failures.", p.Name, p.Failures),
		"color":       0xFF0000,
		"fields": []map[string]interface{}{
			{"name": "Target", "value": p.URL, "inline": false},
			{"name": "Cause", "value": p.Reason, "inline": false},
			{"name": "Failures", "value": fmt.Sprintf("%d", p.Failures), "inline": true},
			{"name": "Timestamp", "value": p.Timestamp.UTC().Format(time.RFC3339), "inline": true},
		},
		"footer": map[string]string{"text": "Pulse99 Uptime Monitor"},
	}
	payload := map[string]interface{}{"embeds": []interface{}{embed}}
	return d.post(payload)
}

func (d *DiscordNotifier) SendRecovery(p Payload) error {
	if d.webhookURL == "" {
		return nil
	}
	embed := map[string]interface{}{
		"title":       fmt.Sprintf("Recovered: %s is ONLINE", p.Name),
		"description": fmt.Sprintf("Target **%s** has recovered.", p.Name),
		"color":       0x00FF00,
		"fields": []map[string]interface{}{
			{"name": "Target", "value": p.URL, "inline": false},
			{"name": "Latency", "value": fmt.Sprintf("%dms", p.LatencyMs), "inline": true},
			{"name": "Timestamp", "value": p.Timestamp.UTC().Format(time.RFC3339), "inline": true},
		},
		"footer": map[string]string{"text": "Pulse99 Uptime Monitor"},
	}
	payload := map[string]interface{}{"embeds": []interface{}{embed}}
	return d.post(payload)
}

func (d *DiscordNotifier) post(payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := d.client.Post(d.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
