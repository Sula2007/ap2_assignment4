package provider

import (
	"errors"
	"log"
	"math/rand"
	"time"
)

type MockEmailSender struct {
	failureRate int
}

func NewMockEmailSender() EmailSender {
	return &MockEmailSender{
		failureRate: 80,
	}
}

func (m *MockEmailSender) Send(to, subject, body string) error {
	time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond)

	if rand.Intn(100) < m.failureRate {
		log.Printf("[MockEmailSender] Simulated failure (rate: %d%%)", m.failureRate)
		return errors.New("mock: simulated provider failure")
	}

	log.Printf("[MockEmailSender] Email sent to=%s subject=%q", to, subject)
	return nil
}