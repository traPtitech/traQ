package search

import (
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"time"
)

// ErrServiceUnavailable エラー 現在検索サービスが利用できません
var ErrServiceUnavailable = errors.New("search service is unavailable")

// Engine 検索エンジンインターフェイス
type Engine interface {
	// Do 与えられたクエリで検索を実行します
	Do(q *Query) (Result, error)
	// Available 検索サービスが利用可能かどうかを返します
	Available() bool
	// Close 検索サービスを終了します
	Close() error
}

// Query 検索クエリ TODO
type Query struct {
	// Word 検索ワード (仮置き)
	Words       string    // 検索ワード TODO 空白区切り(複数) OR NOT (現状1単語)
	After       time.Time // 以降(投稿日時)
	Before      time.Time // 以前(投稿日時)
	To          uuid.UUID // メンション先
	From        uuid.UUID // 投稿者
	Cite        uuid.UUID // 引用しているメッセージ TODO 空白区切り(複数)
	IsEdited    bool
	IsCited     bool
	IsPinned    bool
	HasURL      bool
	HasEmbedded bool
	HasImage    bool
	HasMovie    bool
	HasAudio    bool
}

// Result 検索結果インターフェイス TODO
type Result interface {
	// Get 仮置き
	Get() map[uuid.UUID]string
	// GetMessages() (ms []*model.Message, more bool)
}

// validateもここでする？
func GetSearchQuery(c echo.Context) *Query {
	query := &Query{}

	// words := strings.Split(c.QueryParam("word"), " ")
	query.Words = c.QueryParam("words")

	layout := "2006/1/2 15:04:05"
	query.After, _ = time.Parse(layout, c.QueryParam("after"))
	fmt.Println(time.Parse(layout, c.QueryParam("after")))
	query.Before, _ = time.Parse(layout, c.QueryParam("before"))

	query.To = uuid.FromStringOrNil(c.QueryParam("to"))
	query.From = uuid.FromStringOrNil(c.QueryParam("from"))

	if c.QueryParam("isEdited") == "true" {
		query.IsEdited = true
	}
	if c.QueryParam("isCited") == "true" {
		query.IsCited = true
	}
	if c.QueryParam("isPinned") == "true" {
		query.IsPinned = true
	}

	if c.QueryParam("hasURL") == "true" {
		query.HasURL = true
	}
	if c.QueryParam("hasEmbedded") == "true" {
		query.HasEmbedded = true
	}
	if c.QueryParam("hasImage") == "true" {
		query.HasImage = true
	}
	if c.QueryParam("hasMovie") == "true" {
		query.HasMovie = true
	}
	if c.QueryParam("hasAudio") == "true" {
		query.HasAudio = true
	}

	return query
}
