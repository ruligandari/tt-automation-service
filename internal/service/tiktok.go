package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type TiktokService interface {
	FetchVideo(ctx context.Context, url string) (string, error)
}

type tiktokService struct {
	client       *http.Client
	rapidApiHost string
	rapidApiKeys []string
}

func NewTiktokService(client *http.Client) TiktokService {
	host := os.Getenv("RAPIDAPI_HOST")
	keysEnv := os.Getenv("RAPIDAPI_KEYS")
	var keys []string
	if keysEnv != "" {
		keys = strings.Split(keysEnv, ",")
	}

	return &tiktokService{
		client:       client,
		rapidApiHost: host,
		rapidApiKeys: keys,
	}
}

// response structure matching data.play
type tiktokAPIResponse struct {
	Data struct {
		Play string `json:"play"`
	} `json:"data"`
}

func (s *tiktokService) FetchVideo(ctx context.Context, url string) (string, error) {
	if s.rapidApiHost == "" {
		return "", errors.New("RAPIDAPI_HOST is missing")
	}
	if len(s.rapidApiKeys) == 0 {
		return "", errors.New("RAPIDAPI_KEYS are missing")
	}

	apiEndpoint := fmt.Sprintf("https://%s/?url=%s", s.rapidApiHost, url)

	var lastErr error
	for _, key := range s.rapidApiKeys {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiEndpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("X-RapidAPI-Host", s.rapidApiHost)
		req.Header.Set("X-RapidAPI-Key", strings.TrimSpace(key))

		log.Printf("Trying TikTok Scraper API with key: %s...", key[:minVal(4, len(key))]+"***")
		
		resp, err := s.client.Do(req)
		if err != nil {
			log.Printf("Request failed: %v", err)
			lastErr = err
			continue // try next key
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("API returned non-200 status: %d", resp.StatusCode)
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP status %d", resp.StatusCode)
			continue // try next key
		}

		var apiResp tiktokAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			log.Printf("Failed to decode response: %v", err)
			resp.Body.Close()
			lastErr = err
			continue
		}
		resp.Body.Close()

		if apiResp.Data.Play != "" {
			return apiResp.Data.Play, nil // Success
		}

		log.Println("Response missing data.play field")
		lastErr = errors.New("missing data.play in response")
	}

	return "", fmt.Errorf("all API keys failed. Last error: %w", lastErr)
}

func minVal(a, b int) int {
	if a < b {
		return a
	}
	return b
}
