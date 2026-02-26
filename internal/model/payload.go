package model

// WebhookPayload represents the inner simplified webhook payload
type WebhookPayload struct {
	SessionId string `json:"sessionId"`
	Data      struct {
		Content     string `json:"content"`
		FullMessage struct {
			Key struct {
				SenderPn string `json:"senderPn"`
			} `json:"key"`
		} `json:"fullMessage"`
	} `json:"data"`
}

// WebhookWrapper covers cases where the payload is nested under "body"
type WebhookWrapper struct {
	Body WebhookPayload `json:"body"`
}
