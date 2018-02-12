package notification

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	fcmEndPoint            = "https://fcm.googleapis.com/fcm/send"
	PriorityHigh           = "high"
	PriorityNormal         = "normal"
	MaxRegistrationIdsSize = 1000
)

type FCMClient struct {
	APIKey     string
	HttpClient *http.Client
}

type FCMNotificationPayload struct {
	Title       string `json:"title,omitempty"`
	Body        string `json:"body,omitempty"`
	ClickAction string `json:"click_action,omitempty"`
}

type FCMMessage struct {
	RegistrationIds  []string                `json:"registration_ids,omitempty"`
	Notification     *FCMNotificationPayload `json:"notification,omitempty"`
	Data             interface{}             `json:"data,omitempty"`
	Priority         string                  `json:"priority,omitempty"`
	ContentAvailable bool                    `json:"content_available,omitempty"`
	DryRun           bool                    `json:"dry_run,omitempty"`
}

type FCMResponse struct {
	StatusCode   int
	RetryAfter   string
	MulticastId  int64       `json:"multicast_id"`
	Success      int         `json:"success"`
	Failure      int         `json:"failure"`
	CanonicalIds int         `json:"canonical_ids"`
	Results      []FCMResult `json:"results"`
}

type FCMResult struct {
	MessageId      string `json:"message_id"`
	RegistrationId string `json:"registration_id"`
	Error          string `json:"error"`
}

func NewFCMClient(apiKey string) *FCMClient {
	return &FCMClient{
		APIKey:     apiKey,
		HttpClient: &http.Client{},
	}
}

func (c *FCMClient) Send(message *FCMMessage) (*FCMResponse, error) {
	data, err := message.Marshal()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fcmEndPoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("key=%v", c.APIKey))
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusUnauthorized {
			panic(errors.New("firebase authentication failed. the api key is invalid"))
		} else {
			return nil, fmt.Errorf("fcm error: %d - %s", res.StatusCode, res.Status)
		}
	}

	r := &FCMResponse{}
	r.StatusCode = res.StatusCode
	r.RetryAfter = res.Header.Get("Retry-After")
	if err := json.NewDecoder(res.Body).Decode(r); err != nil {
		return nil, err
	}

	return r, nil
}

func (m *FCMMessage) Marshal() ([]byte, error) {
	//TODO Validation
	return json.Marshal(m)
}

func (r FCMResult) Unregistered() bool {
	switch r.Error {
	case "MismatchSenderId", "NotRegistered", "InvalidRegistration", "MissingRegistration":
		return true
	default:
		return false
	}
}

func (r *FCMResponse) IsTimeout() bool {
	if r.StatusCode >= 500 {
		return true
	} else if r.StatusCode == 200 {
		for _, v := range r.Results {
			if v.Error == "Unavailable" || v.Error == "InternalServerError" {
				return true
			}
		}
	}

	return false
}

func (r *FCMResponse) GetInvalidRegistration() []string {
	var ids []string
	for _, v := range r.Results {
		if v.Unregistered() {
			ids = append(ids, v.RegistrationId)
		}
	}
	return ids
}

func (r *FCMResponse) GetRetryAfterTime() (time.Duration, error) {
	return time.ParseDuration(r.RetryAfter)
}
