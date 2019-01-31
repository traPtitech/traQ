//+build tools

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	webhookUrl := os.Getenv("STAGING_DEPLOY_WEBHOOK_URL")
	secret := os.Getenv("STAGING_DEPLOY_SECRET")
	repoUser := os.Getenv("CIRCLE_PROJECT_USERNAME")
	repoName := os.Getenv("CIRCLE_PROJECT_REPONAME")
	branch := os.Getenv("CIRCLE_BRANCH")

	payload := struct {
		Ref        string `json:"ref"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}{}
	payload.Ref = "refs/heads/" + branch
	payload.Repository.FullName = repoUser + "/" + repoName

	b, err := json.Marshal(&payload)
	if err != nil {
		log.Fatal(err)
	}

	mac := hmac.New(sha1.New, []byte(secret))
	if _, err := mac.Write(b); err != nil {
		log.Fatal(err)
	}
	sig := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, webhookUrl, bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature", fmt.Sprintf("sha1=%s", sig))

	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()
}
