package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	
	"tiktok-automation-service/internal/model"
	"tiktok-automation-service/internal/service"
	"tiktok-automation-service/pkg/response"
)

type WebhookHandler struct {
	tiktokService   service.TiktokService
	whatsappService service.WhatsAppService
}

func NewWebhookHandler(ts service.TiktokService, ws service.WhatsAppService) *WebhookHandler {
	return &WebhookHandler{
		tiktokService:   ts,
		whatsappService: ws,
	}
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Always return HTTP 200 to prevent webhook retry
	defer response.SendOK(w)

	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s", r.Method)
		return
	}

	// 1. Parse JSON body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		return
	}
	defer r.Body.Close()

	var payload model.WebhookPayload

	// Try unmarshaling as an array of wrappers
	var arrayPayload []model.WebhookWrapper
	if err := json.Unmarshal(bodyBytes, &arrayPayload); err == nil && len(arrayPayload) > 0 {
		payload = arrayPayload[0].Body
	} else {
		// Fallback to single wrapper
		var singlePayload model.WebhookWrapper
		if err := json.Unmarshal(bodyBytes, &singlePayload); err == nil && singlePayload.Body.SessionId != "" {
			payload = singlePayload.Body
		} else {
			// Fallback to direct raw payload
			if err := json.Unmarshal(bodyBytes, &payload); err != nil {
				log.Printf("Failed to decode JSON block: %v", err)
				return
			}
		}
	}

	// Extract and log fields
	sessionId := payload.SessionId
	content := payload.Data.Content
	senderPn := payload.Data.FullMessage.Key.SenderPn

	log.Printf("Received webhook - Session: %s, Sender: %s, Content: %s", sessionId, senderPn, content)

	// 2. Validate sessionId
	expectedSessionId := os.Getenv("SESSION_ID")
	if expectedSessionId != "" && sessionId != expectedSessionId {
		log.Printf("Ignoring webhook: sessionId mismatch (expected: %s, got: %s)", expectedSessionId, sessionId)
		return
	}

	// 3. Check message contains "https://"
	if !strings.Contains(content, "https://") {
		log.Printf("Ignoring webhook: message does not contain https:// link")
		return
	}
	
	log.Println("Webhook validation passed, ready to process video link")

	// Extract URL from content using Regex
	// Support regular tiktok.com and shortened vt.tiktok.com
	tiktokRegex := `https?://(?:www\.|vt\.)?tiktok\.com/[a-zA-Z0-9/_?=&%-]+`
	re := regexp.MustCompile(tiktokRegex)
	targetURL := re.FindString(content)

	if targetURL == "" {
		log.Println("No valid TikTok URL found in content")
		return
	}

	// 4. Fetch video URL via tiktokService
	log.Printf("Fetching TikTok video for URL: %s", targetURL)
	videoURL, err := h.tiktokService.FetchVideo(r.Context(), targetURL)
	if err != nil {
		log.Printf("Failed to fetch TikTok video: %v", err)
		return
	}

	log.Printf("Successfully fetched video URL: %s", videoURL)

	// 5. Send media via whatsappService
	caption := "ini bosque, video dari: " + targetURL

	// Make sure we have a senderPn
	if senderPn == "" {
		log.Println("Cannot send WhatsApp message: senderPn is empty")
		return
	}

	// Remove suffix like "@s.whatsapp.net" or "@lid" if needed, 
	// but normally the sender API accepts the raw JID or just the number.
	// For standard API implementations, we typically need clean number or let the API handle it.
	// The blueprint doesn't specify stripping, but a wa gateway normally requires the full jid or just the digits.
	// We will pass senderPn as is to SendMedia.
	log.Printf("Sending video back to %s...", senderPn)
	if err := h.whatsappService.SendMedia(r.Context(), senderPn, videoURL, caption); err != nil {
		log.Printf("Failed to send media via WhatsApp: %v", err)
		return
	}

	log.Println("Webhook processed completely and successfully.")
}
