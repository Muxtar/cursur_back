package utils

import (
	"fmt"
	"log"

	"chat-backend/internal/config"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

type TwilioService struct {
	client   *twilio.RestClient
	from     string
	enabled  bool
}

func NewTwilioService(cfg *config.Config) *TwilioService {
	if !cfg.TwilioEnabled || cfg.TwilioAccountSID == "" || cfg.TwilioAuthToken == "" || cfg.TwilioPhoneNumber == "" {
		log.Println("Twilio is disabled or not configured. SMS will not be sent.")
		return &TwilioService{
			enabled: false,
		}
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.TwilioAccountSID,
		Password: cfg.TwilioAuthToken,
	})

	return &TwilioService{
		client:  client,
		from:    cfg.TwilioPhoneNumber,
		enabled: true,
	}
}

func (ts *TwilioService) SendSMS(to, message string) error {
	if !ts.enabled {
		return fmt.Errorf("Twilio is not enabled or configured")
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(ts.from)
	params.SetBody(message)

	resp, err := ts.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	log.Printf("SMS sent successfully. SID: %s, Status: %s\n", *resp.Sid, *resp.Status)
	return nil
}

func (ts *TwilioService) SendVerificationCode(phoneNumber, code string) error {
	message := fmt.Sprintf("Your verification code is: %s. This code will expire in 5 minutes.", code)
	return ts.SendSMS(phoneNumber, message)
}

func (ts *TwilioService) IsEnabled() bool {
	return ts.enabled
}
