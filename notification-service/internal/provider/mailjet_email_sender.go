package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

type MailjetEmailSender struct {
	apiKey      string
	secretKey   string
	senderEmail string
	httpClient  *http.Client
}

func NewMailjetEmailSender(apiKey, secretKey, senderEmail string) EmailSender {
	return &MailjetEmailSender{
		apiKey:      apiKey,
		secretKey:   secretKey,
		senderEmail: senderEmail,
		httpClient:  &http.Client{Timeout: 10 * http.DefaultClient.Timeout},
	}
}

type mailjetMessage struct {
	From struct {
		Email string `json:"Email"`
	} `json:"From"`
	To []struct {
		Email string `json:"Email"`
	} `json:"To"`
	Subject  string `json:"Subject"`
	TextPart string `json:"TextPart"`
}

type mailjetRequest struct {
	Messages []mailjetMessage `json:"Messages"`
}

func (m *MailjetEmailSender) Send(to, subject, body string) error {
	msg := mailjetMessage{}
	msg.From.Email = m.senderEmail
	msg.To = []struct {
		Email string `json:"Email"`
	}{{Email: to}}
	msg.Subject = subject
	msg.TextPart = body

	payload := mailjetRequest{Messages: []mailjetMessage{msg}}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mailjet marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.mailjet.com/v3.1/send", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("mailjet new request: %w", err)
	}

	creds := base64.StdEncoding.EncodeToString([]byte(m.apiKey + ":" + m.secretKey))
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mailjet send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("mailjet error status: %d", resp.StatusCode)
	}

	return nil
}
