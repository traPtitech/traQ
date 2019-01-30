package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/oauth2"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	botResponseContentLimit        = 10 * 1024 // bytes
	eventActivation                = "ACTIVATION"
	headerTRAQBotEvent             = "X-TRAQ-BOT-EVENT"
	headerTRAQBotRequestID         = "X-TRAQ-BOT-REQUEST-ID"
	headerTRAQBotVerificationToken = "X-TRAQ-BOT-TOKEN"
	headerTRAQBotEventDateTime     = "X-TRAQ-BOT-EVENT-DATETIME"
)

var (
	// ErrBotNotFound Botが見つかりません
	ErrBotNotFound = errors.New("not found")
	// ErrBotActivationFailed Botのアクティベーションに失敗しました
	ErrBotActivationFailed = errors.New("activation failed")
)

type eventPayload interface {
	GetBotPayload() interface{}
}

type eventTarget interface {
	GetTargetChannels() map[uuid.UUID]bool
}

// BotProcessor Bot Processor
type BotProcessor struct {
	oauth2            *oauth2.Handler
	activationStarted sync.Map
	botReqClient      http.Client
}

// NewBotProcessor BotDaoを作成します
func NewBotProcessor(oauth2 *oauth2.Handler) *BotProcessor {
	dao := &BotProcessor{
		oauth2: oauth2,
		botReqClient: http.Client{
			Timeout: 10 * time.Second,
		},
	}
	return dao
}

// Process イベントを処理します
func (h *BotProcessor) Process(t Type, time time.Time, d interface{}) error {
	data, ok := d.(eventPayload)
	if !ok {
		return nil
	}
	payload := data.GetBotPayload()
	targetCID := uuid.Nil
	if v, ok := d.(eventTarget); ok {
		for k := range v.GetTargetChannels() {
			targetCID = k
		}
	}

	switch t {
	case MessageCreated, MessageUpdated, MessageDeleted:
		bots, err := model.GetInstalledBots(targetCID)
		if err != nil {
			return err
		}
		for _, v := range bots {
			go h.checkAndSend(v.GetBID(), time, string(t), payload)
		}
	}

	return nil
}

func (h *BotProcessor) checkAndSend(botID uuid.UUID, time time.Time, e string, payload interface{}) {
	b, _ := model.GetBot(botID)
	if b != nil && b.GetActivated() && b.GetSubscribeEvents()[e] {
		_, _ = h.sendEventToBot(b, time, e, payload)
	}
}

func (h *BotProcessor) sendEventToBot(b model.Bot, time time.Time, e string, data interface{}) (uuid.UUID, error) {
	reqID := uuid.NewV4()

	// Jsonリクエスト構築
	dataStr, r := makePayload(data)
	req, _ := http.NewRequest(http.MethodPost, b.GetPostURL().String(), r)
	if r != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	}
	req.Header.Set(headerTRAQBotEvent, e)
	req.Header.Set(headerTRAQBotRequestID, reqID.String())
	req.Header.Set(headerTRAQBotVerificationToken, b.GetVerificationToken())
	req.Header.Set(headerTRAQBotEventDateTime, time.String())

	reqSum := summarizeRequest(req, dataStr)
	res, err := h.botReqClient.Do(req) //タイムアウトは10秒
	if err != nil {
		// ネットワークエラー
		_ = model.SavePostLog(reqID, b.GetID(), 0, reqSum, "", err.Error())
		return reqID, err
	}

	resSummary := &strings.Builder{}
	resSummary.WriteString(fmt.Sprintf("HTTP/1.1 %s\n", res.Status))
	writeHTTPHeaders(resSummary, res.Header)

	resBody, err := ioutil.ReadAll(io.LimitReader(res.Body, botResponseContentLimit+1))
	res.Body.Close()
	if err != nil {
		// ストリームエラー
		_ = model.SavePostLog(reqID, b.GetID(), res.StatusCode, reqSum, resSummary.String(), err.Error())
		return reqID, err
	}

	// レスポンスサイズ制限 (Content-Lengthヘッダを用いてはいけない(不定の場合があるため))
	if len(resBody) > botResponseContentLimit {
		_ = model.SavePostLog(reqID, b.GetID(), res.StatusCode, reqSum, resSummary.String(), "too big response")
		return reqID, errors.New("too big response")
	}

	resSummary.WriteString("\n")
	resSummary.Write(resBody)
	_ = model.SavePostLog(reqID, b.GetID(), res.StatusCode, reqSum, resSummary.String(), "")

	// ステータスコードがOK以外の場合は無効
	if res.StatusCode != http.StatusOK {
		return reqID, errors.New("the bot didn't return StatusOK")
	}

	// レスポンスに内容が含まれているか
	// if len(resBody) > 0 {
	// TODO 直接メッセージ投稿などが出来る様にする
	// }

	return reqID, nil
}

// ActivateBot Botのアクティベーションを行います
func (h *BotProcessor) ActivateBot(id uuid.UUID) error {
	b, err := model.GetBot(id)
	if err != nil {
		return err
	}
	if b == nil {
		return ErrBotNotFound
	}

	if _, ok := h.activationStarted.LoadOrStore(id, struct{}{}); ok {
		return errors.New("the bot activation process has already started")
	}
	defer h.activationStarted.Delete(id)

	if _, err := h.sendEventToBot(b, time.Now(), eventActivation, nil); err != nil {
		return ErrBotActivationFailed
	}

	if err := model.ActivateBot(b.GetID()); err != nil {
		return err
	}
	return nil
}

func makePayload(data interface{}) (string, io.Reader) {
	payload := &strings.Builder{}
	payload.Grow(2 << 10) // 1KB確保
	if data != nil {
		if err := json.NewEncoder(payload).Encode(data); err != nil {
			panic(err) // 変なデータが流れてくるのは実装がおかしいからパニック
		}
	}

	// payloadが無い場合はio.Readerはnilを返す
	var r io.Reader
	if payload.Len() > 0 {
		r = strings.NewReader(payload.String()) //アロケーションは発生しない(と思う)
	}

	return payload.String(), r
}

func summarizeRequest(req *http.Request, payload string) string {
	s := &strings.Builder{}
	s.Grow(len(payload))
	s.WriteString("POST ")
	s.WriteString(req.URL.String())
	s.WriteString(" HTTP/1.1\n")
	writeHTTPHeaders(s, req.Header)
	s.WriteString("\n")
	s.WriteString(payload)
	return s.String()
}

func writeHTTPHeaders(builder *strings.Builder, headers http.Header) {
	for k, vs := range headers {
		for _, v := range vs {
			builder.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}
}
