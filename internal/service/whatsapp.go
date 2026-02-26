package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
)

type WhatsAppService interface {
	SendMedia(ctx context.Context, number string, mediaURL string, caption string) error
}

type whatsappService struct {
	client    *http.Client
	apiURL    string
	apiKey    string
	sessionId string
}

func NewWhatsAppService(client *http.Client) WhatsAppService {
	return &whatsappService{
		client:    client,
		apiURL:    os.Getenv("WA_API_URL"),
		apiKey:    os.Getenv("WA_API_KEY"),
		sessionId: os.Getenv("SESSION_ID"),
	}
}

// whatsappPayload represents the specific payload for the provided WhatsApp Gateway API
type whatsappPayload struct {
	Number    string `json:"number"`
	MediaUrl  string `json:"mediaUrl"`
	Caption   string `json:"caption,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
}

func (s *whatsappService) SendMedia(ctx context.Context, number string, mediaURL string, caption string) error {
	if s.apiURL == "" {
		return errors.New("WA_API_URL is missing")
	}
	if s.sessionId == "" {
		return errors.New("SESSION_ID is missing")
	}

	payload := whatsappPayload{
		Number:    number,
		MediaUrl:  mediaURL,
		Caption:   caption,
		MediaType: "video",
	}

	jsonValue, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Format endpoint: e.g. https://wa-sender.hellodev.my.id/api/whatsapp/session/ruli2/send-media
	baseURL := s.apiURL
	if baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	endpoint := fmt.Sprintf("%s/api/whatsapp/session/%s/send-media", baseURL, s.sessionId)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonValue))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.apiKey != "" {
		req.Header.Set("x-api-key", s.apiKey)
	}

	log.Printf("Sending media to WhatsApp API (number: %s)...", number)
	
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request to WA API failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("WA API returned error status: %d", resp.StatusCode)
	}

	log.Printf("Successfully sent media to %s", number)
	return nil
}
