package bot

import (
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	botResponseContentLimit        = 5 * 1024 * 1024 // bytes
	headerTRAQBotEvent             = "X-TRAQ-BOT-EVENT"
	headerTRAQBotRequestID         = "X-TRAQ-BOT-REQUEST-ID"
	headerTRAQBotVerificationToken = "X-TRAQ-BOT-Token"
)

type bot interface {
	GetPostURL() string
	GetVerificationToken() string
	GetBotUserID() uuid.UUID
}

func (h *Dao) sendEventToBot(b bot, event event.Type, payload string) (uuid.UUID, error) {
	reqID := uuid.NewV4()

	// リクエスト構築
	var r io.Reader = nil
	if len(payload) > 0 {
		r = strings.NewReader(payload)
	}
	req, _ := http.NewRequest(http.MethodPost, b.GetPostURL(), r)
	req.Header.Set(headerTRAQBotEvent, string(event))
	req.Header.Set(headerTRAQBotRequestID, reqID.String())
	req.Header.Set(headerTRAQBotVerificationToken, b.GetVerificationToken())

	res, err := h.botReqClient.Do(req)

	reqSummary := &strings.Builder{}
	reqSummary.WriteString(fmt.Sprintf("POST %s HTTP/1.1\n", b.GetPostURL()))
	writeHttpHeaders(reqSummary, req.Header)
	reqSummary.WriteString("\n")
	reqSummary.WriteString(payload)

	if err != nil {
		// ネットワークエラー
		h.store.SavePostLog(reqID, b.GetBotUserID(), 0, reqSummary.String(), "", err.Error())
		return reqID, err
	}
	defer res.Body.Close()

	resSummary := &strings.Builder{}
	resSummary.WriteString(fmt.Sprintf("HTTP/1.1 %s\n", res.Status))
	writeHttpHeaders(resSummary, res.Header)

	resBody, err := ioutil.ReadAll(io.LimitReader(res.Body, botResponseContentLimit+1))
	if err != nil {
		// ストリームエラー
		h.store.SavePostLog(reqID, b.GetBotUserID(), res.StatusCode, reqSummary.String(), resSummary.String(), err.Error())
		return reqID, err
	}

	// 5MB制限 (Content-Lengthヘッダを用いてはいけない(不定の場合があるため))
	if len(resBody) > botResponseContentLimit {
		h.store.SavePostLog(reqID, b.GetBotUserID(), res.StatusCode, reqSummary.String(), resSummary.String(), "too big response")
		return reqID, errors.New("too big response")
	}

	resSummary.WriteString("\n")
	resSummary.Write(resBody)
	h.store.SavePostLog(reqID, b.GetBotUserID(), res.StatusCode, reqSummary.String(), resSummary.String(), "")

	// ステータスコードがOK以外の場合は無効
	if res.StatusCode != http.StatusOK {
		return reqID, errors.New("the bot didn't return StatusOK")
	}

	// レスポンスに内容が含まれているか
	if len(resBody) > 0 {
		//TODO
	}

	return reqID, nil
}

func writeHttpHeaders(builder *strings.Builder, headers http.Header) {
	for k, vs := range headers {
		for _, v := range vs {
			builder.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}
}
