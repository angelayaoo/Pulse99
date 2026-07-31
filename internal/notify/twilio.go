package notify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TwilioNotifier struct {
	accountSID string
	authToken  string
	fromPhone  string
	toPhones   []string
	client     *http.Client
}

func NewTwilio(accountSID, authToken, fromPhone string, toPhones []string) *TwilioNotifier {
	return &TwilioNotifier{
		accountSID: accountSID,
		authToken:  authToken,
		fromPhone:  fromPhone,
		toPhones:   toPhones,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TwilioNotifier) Name() string { return "twilio" }

func (t *TwilioNotifier) SendCritical(p Payload) error {
	if t.accountSID == "" || t.authToken == "" || len(t.toPhones) == 0 {
		return nil
	}
	body := fmt.Sprintf(
		"ALERT: %s DOWN\nFailures: %d\n%s\n-- Pulse99",
		p.Name, p.Failures, p.Reason,
	)
	return t.sendSMS(body)
}

func (t *TwilioNotifier) SendRecovery(p Payload) error {
	if t.accountSID == "" || t.authToken == "" || len(t.toPhones) == 0 {
		return nil
	}
	body := fmt.Sprintf(
		"RECOVERED: %s ONLINE (%dms)\n-- Pulse99",
		p.Name, p.LatencyMs,
	)
	return t.sendSMS(body)
}

func (t *TwilioNotifier) sendSMS(body string) error {
	for _, phone := range t.toPhones {
		apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)
		data := url.Values{}
		data.Set("From", t.fromPhone)
		data.Set("To", phone)
		data.Set("Body", body)

		req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
		if err != nil {
			return err
		}

		auth := base64.StdEncoding.EncodeToString([]byte(t.accountSID + ":" + t.authToken))
		req.Header.Set("Authorization", "Basic "+auth)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := t.client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
	}
	return nil
}
