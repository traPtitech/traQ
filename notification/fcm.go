package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	fcmEndPointTemplate    = "https://fcm.googleapis.com/v1/projects/{PROJECT_ID}/messages:send"
	fcmOAuth2Scope         = "https://www.googleapis.com/auth/firebase.messaging"
	priorityHigh           = "high"
	priorityNormal         = "normal"
	maxRegistrationIdsSize = 1000
)

type fcmClient struct {
	credentials *google.Credentials
	endpoint    string
	HTTPClient  *http.Client
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

func newFCMClient(serviceAccountJsonFile string) *fcmClient {
	ctx := context.Background()
	data, err := ioutil.ReadFile(serviceAccountJsonFile)
	if err != nil {
		panic(err)
	}
	creds, err := google.CredentialsFromJSON(ctx, data, fcmOAuth2Scope)
	if err != nil {
		panic(err)
	}

	c := &fcmClient{
		credentials: creds,
		endpoint:    strings.Replace(fcmEndPointTemplate, "{PROJECT_ID}", creds.ProjectID, 1),
		HTTPClient:  &http.Client{},
	}
	return c
}

func (c *fcmClient) send(message *fcmMessage) (*fcmResponse, error) {
	data, err := message.marshal()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	token, err := c.credentials.TokenSource.Token()
	if err != nil {
		return nil, err
	}

	token.SetAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
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
