package provider

import (
	"errors"
	"log"
	"math/rand"
	"time"
)

type MockEmailSender struct{}

func NewMockEmailSender() EmailSender {
	return &MockEmailSender{}
}

func (m *MockEmailSender) Send(to, subject, body string) error {
	time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)

	if rand.Intn(4) == 0 {
		return errors.New("mock: simulated provider failure")
	}

	log.Printf("[MockEmailSender] Email sent to=%s subject=%q", to, subject)
	return nil
}
