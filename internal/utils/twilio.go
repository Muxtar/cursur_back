package utils

import (
	"fmt"
	"log"
	"strings"

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

// normalizePhoneNumber ensures phone number is in E.164 format (+1234567890)
func normalizePhoneNumber(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return phone
	}
	// Remove all non-digit characters except +
	normalized := strings.Builder{}
	if strings.HasPrefix(phone, "+") {
		normalized.WriteString("+")
		phone = phone[1:]
	}
	// Add only digits
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			normalized.WriteRune(char)
		}
	}
	return normalized.String()
}

func (ts *TwilioService) SendSMS(to, message string) error {
	if !ts.enabled {
		return fmt.Errorf("Twilio is not enabled or configured")
	}

	if to == "" {
		return fmt.Errorf("recipient phone number is required")
	}

	if message == "" {
		return fmt.Errorf("message body is required")
	}

	// Normalize phone numbers to E.164 format
	normalizedTo := normalizePhoneNumber(to)
	if normalizedTo == "" || !strings.HasPrefix(normalizedTo, "+") {
		return fmt.Errorf("invalid phone number format. Phone number must be in E.164 format (e.g., +18777804236)")
	}

	normalizedFrom := normalizePhoneNumber(ts.from)
	if normalizedFrom == "" || !strings.HasPrefix(normalizedFrom, "+") {
		return fmt.Errorf("invalid Twilio phone number format. Must be in E.164 format")
	}

	// Create message parameters (equivalent to curl --data-urlencode)
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(normalizedTo)      // Recipient phone number (e.g., +18777804236)
	params.SetFrom(normalizedFrom)   // Sender phone number (Twilio phone number)
	params.SetBody(message)          // Message content

	// Send SMS via Twilio API
	resp, err := ts.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("failed to send SMS to %s: %w", normalizedTo, err)
	}

	// Check response
	if resp.Sid != nil {
		log.Printf("SMS sent successfully. SID: %s, To: %s, From: %s, Status: %s\n", 
			*resp.Sid, normalizedTo, normalizedFrom, getStatus(resp))
	} else {
		log.Printf("SMS sent but no SID returned. To: %s, From: %s\n", normalizedTo, normalizedFrom)
	}

	return nil
}

// Helper function to safely get status from response
func getStatus(resp *twilioApi.ApiV2010Message) string {
	if resp.Status != nil {
		return string(*resp.Status)
	}
	return "unknown"
}

func (ts *TwilioService) SendVerificationCode(phoneNumber, code string) error {
	message := fmt.Sprintf("Your verification code is: %s. This code will expire in 5 minutes.", code)
	return ts.SendSMS(phoneNumber, message)
}

func (ts *TwilioService) IsEnabled() bool {
	return ts.enabled
}
