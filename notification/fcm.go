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
	priorityHigh           = "high"
	priorityNormal         = "normal"
	maxRegistrationIdsSize = 1000
)

type fcmClient struct {
	APIKey     string
	HttpClient *http.Client
}

type fcmNotificationPayload struct {
	Title       string `json:"title,omitempty"`
	Body        string `json:"body,omitempty"`
	ClickAction string `json:"click_action,omitempty"`
}

type fcmMessage struct {
	RegistrationIDs  []string                `json:"registration_ids,omitempty"`
	Notification     *fcmNotificationPayload `json:"notification,omitempty"`
	Data             interface{}             `json:"data,omitempty"`
	Priority         string                  `json:"priority,omitempty"`
	ContentAvailable bool                    `json:"content_available,omitempty"`
	DryRun           bool                    `json:"dry_run,omitempty"`
}

type fcmResponse struct {
	StatusCode   int
	RetryAfter   string
	MulticastID  int64       `json:"multicast_id"`
	Success      int         `json:"success"`
	Failure      int         `json:"failure"`
	CanonicalIDs int         `json:"canonical_ids"`
	Results      []fcmResult `json:"results"`
}

type fcmResult struct {
	MessageID      string `json:"message_id"`
	RegistrationID string `json:"registration_id"`
	Error          string `json:"error"`
}

func newFCMClient(apiKey string) *fcmClient {
	return &fcmClient{
		APIKey:     apiKey,
		HttpClient: &http.Client{},
	}
}

func (c *fcmClient) send(message *fcmMessage) (*fcmResponse, error) {
	data, err := message.marshal()
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

	r := &fcmResponse{}
	r.StatusCode = res.StatusCode
	r.RetryAfter = res.Header.Get("Retry-After")
	if err := json.NewDecoder(res.Body).Decode(r); err != nil {
		return nil, err
	}

	return r, nil
}

func (m *fcmMessage) marshal() ([]byte, error) {
	//TODO Validation
	return json.Marshal(m)
}

func (r fcmResult) unregistered() bool {
	switch r.Error {
	case "MismatchSenderId", "NotRegistered", "InvalidRegistration", "MissingRegistration":
		return true
	default:
		return false
	}
}

func (r *fcmResponse) isTimeout() bool {
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

func (r *fcmResponse) getInvalidRegistration() []string {
	var ids []string
	for _, v := range r.Results {
		if v.unregistered() {
			ids = append(ids, v.RegistrationID)
		}
	}
	return ids
}

func (r *fcmResponse) getRetryAfterTime() (time.Duration, error) {
	return time.ParseDuration(r.RetryAfter)
}
