package search

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
)

// ImageSearchConfig 画像検索用外部サーバー設定
type ImageSearchConfig struct {
	// URL 外部サーバーのURL (空の場合は画像検索無効)
	URL string
	// Timeout リクエストタイムアウト
	Timeout time.Duration
	// VectorDimension Embeddingベクトルの次元数
	VectorDimension int
}

// ImageProcessResult 外部サーバーによる画像処理結果
type ImageProcessResult struct {
	// Text OCRで抽出されたテキスト
	Text string
	// Vector Embeddingベクトル
	Vector []float64
}

// ImageSearchClient 画像検索用外部サーバークライアントインターフェース
type ImageSearchClient interface {
	// EmbedText テキストをベクトルに変換する（検索クエリ用）
	EmbedText(ctx context.Context, text string) ([]float64, error)
	// ProcessImages 画像URLリストからOCR/Embeddingを実行する
	ProcessImages(ctx context.Context, imageURLs []string) ([]ImageProcessResult, error)
	// Available 外部サーバーが利用可能かどうか
	Available() bool
}

// stubImageSearchClient スタブ実装
type stubImageSearchClient struct {
	config ImageSearchConfig
	l      *zap.Logger
}

// NewImageSearchClient 画像検索クライアントを生成する
// 現在はスタブ実装を返す
func NewImageSearchClient(config ImageSearchConfig, logger *zap.Logger) ImageSearchClient {
	if config.URL == "" {
		return &nullImageSearchClient{}
	}
	return &stubImageSearchClient{
		config: config,
		l:      logger.Named("image-search-client"),
	}
}

func (c *stubImageSearchClient) EmbedText(ctx context.Context, text string) ([]float64, error) {
	// TODO: 外部サーバーにテキストを送信してベクトルを取得する
	// POST {config.URL}/embed/text
	// Request:  { "text": "..." }
	// Response: { "vector": [0.1, 0.2, ...] }
	c.l.Debug("stub: EmbedText called", zap.String("text", text))
	return nil, ErrImageSearchUnavailable
}

func (c *stubImageSearchClient) ProcessImages(ctx context.Context, imageURLs []string) ([]ImageProcessResult, error) {
	// TODO: 外部サーバーに画像の署名付きURLを送信してOCR/Embeddingを取得する
	// POST {config.URL}/process/images
	// Request:  { "imageUrls": ["https://...signed-url..."] }
	// Response: { "results": [{ "text": "OCR結果", "vector": [0.1, ...] }] }
	c.l.Debug("stub: ProcessImages called", zap.Int("imageCount", len(imageURLs)))
	return nil, ErrImageSearchUnavailable
}

func (c *stubImageSearchClient) Available() bool {
	// TODO: 外部サーバーのヘルスチェック
	return false
}

// nullImageSearchClient 画像検索無効時のnull実装
type nullImageSearchClient struct{}

func (c *nullImageSearchClient) EmbedText(context.Context, string) ([]float64, error) {
	return nil, ErrImageSearchUnavailable
}

func (c *nullImageSearchClient) ProcessImages(context.Context, []string) ([]ImageProcessResult, error) {
	return nil, ErrImageSearchUnavailable
}

func (c *nullImageSearchClient) Available() bool {
	return false
}

// ImageAttachment 画像添付ファイル情報
type ImageAttachment struct {
	FileID uuid.UUID
	Key    string
}
