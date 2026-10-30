package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
)

// EmbeddingDimension embedding-serverが返すベクトルの次元数
const EmbeddingDimension = 4096

const (
	maxEmbeddingTextLength = 8192
	maxEmbeddingImageSize  = 20 << 20
)

// ImageSearchConfig 画像検索用外部サーバー設定
type ImageSearchConfig struct {
	// URL 外部サーバーのURL (空の場合は画像検索無効)
	URL string
	// APIKey 公開APIキー
	APIKey string
	// WebhookURL ジョブ完了通知を受け取るURL (空の場合は通知を要求しない)
	WebhookURL string
	// TextTimeout テキスト埋め込みのタイムアウト
	TextTimeout time.Duration
	// JobPollInterval 画像ジョブのポーリング間隔
	JobPollInterval time.Duration
	// JobTimeout 画像ジョブの完了待ち上限
	JobTimeout time.Duration
}

// ImageJobStatus embedding-serverの画像埋め込みジョブ状態
type ImageJobStatus string

const (
	// ImageJobPending 処理待ち
	ImageJobPending ImageJobStatus = "pending"
	// ImageJobProcessing 処理中
	ImageJobProcessing ImageJobStatus = "processing"
	// ImageJobCompleted 完了
	ImageJobCompleted ImageJobStatus = "completed"
	// ImageJobFailed 失敗
	ImageJobFailed ImageJobStatus = "failed"
)

// ImageSearchClient 画像検索用外部サーバークライアントインターフェース
type ImageSearchClient interface {
	// EmbedText テキストをベクトルに変換する（検索クエリ用）
	EmbedText(ctx context.Context, text string) ([]float64, error)
	// SubmitImageJob 画像を送信して埋め込みジョブを作成する
	SubmitImageJob(ctx context.Context, filename, contentType string, r io.Reader, webhookURL string) (uuid.UUID, error)
	// GetJob 埋め込みジョブの状態と結果を取得する
	GetJob(ctx context.Context, jobID uuid.UUID) (ImageJobStatus, []float64, error)
	// Available 外部サーバーが利用可能かどうか
	Available() bool
}

type httpImageSearchClient struct {
	config ImageSearchConfig
	client *http.Client
	l      *zap.Logger
}

// NewImageSearchClient 画像検索クライアントを生成する
func NewImageSearchClient(config ImageSearchConfig, logger *zap.Logger) ImageSearchClient {
	if config.URL == "" {
		return &nullImageSearchClient{}
	}
	return &httpImageSearchClient{
		config: config,
		client: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		l: logger.Named("image-search-client"),
	}
}

func (c *httpImageSearchClient) endpoint(path string) string {
	return strings.TrimRight(c.config.URL, "/") + path
}

func (c *httpImageSearchClient) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), body)
	if err != nil {
		return nil, err
	}
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
	return req, nil
}

type embeddingResult struct {
	Vector []float64 `json:"vector"`
}

type embeddingErrorResponse struct {
	Message string `json:"message"`
}

func (c *httpImageSearchClient) EmbedText(ctx context.Context, text string) ([]float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("embedding text is empty")
	}
	if utf8.RuneCountInString(text) > maxEmbeddingTextLength {
		return nil, fmt.Errorf("embedding text is too long (max %d characters)", maxEmbeddingTextLength)
	}

	timeout := c.config.TextTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	body, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < 2; attempt++ {
		vector, retryAfter, err := c.postText(ctx, body)
		if err != nil {
			return nil, err
		}
		if vector != nil {
			return vector, nil
		}

		// 503の場合はRetry-Afterに従って1回だけ再試行する
		if attempt == 0 {
			if retryAfter > 0 {
				select {
				case <-time.After(retryAfter):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			continue
		}
		return nil, errors.New("embedding server is busy")
	}
	return nil, errors.New("embedding server is busy")
}

// postText テキスト埋め込みを送信する。503の場合はvectorがnil、retryAfterに待機時間を返す。
func (c *httpImageSearchClient) postText(ctx context.Context, body []byte) (vector []float64, retryAfter time.Duration, err error) {
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/embeddings/text", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusServiceUnavailable {
		return nil, parseRetryAfter(res.Header.Get("Retry-After")), nil
	}
	if res.StatusCode != http.StatusOK {
		return nil, 0, c.responseError(res)
	}

	var result embeddingResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("failed to decode embedding response: %w", err)
	}
	if len(result.Vector) != EmbeddingDimension {
		return nil, 0, fmt.Errorf("unexpected embedding dimension: got %d, want %d", len(result.Vector), EmbeddingDimension)
	}
	return result.Vector, 0, nil
}

func (c *httpImageSearchClient) SubmitImageJob(ctx context.Context, filename, contentType string, r io.Reader, webhookURL string) (uuid.UUID, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	if filename == "" {
		filename = "image"
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     "images",
		"filename": filename,
	}))
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := io.Copy(part, r); err != nil {
		return uuid.Nil, err
	}
	if webhookURL != "" {
		if err := writer.WriteField("webhook_url", webhookURL); err != nil {
			return uuid.Nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return uuid.Nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/v1/embeddings/images", &buf)
	if err != nil {
		return uuid.Nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := c.client.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		return uuid.Nil, c.responseError(res)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode embedding job response: %w", err)
	}
	jobID, err := uuid.FromString(result.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid embedding job id: %w", err)
	}
	return jobID, nil
}

func (c *httpImageSearchClient) GetJob(ctx context.Context, jobID uuid.UUID) (ImageJobStatus, []float64, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/embeddings/jobs/"+url.PathEscape(jobID.String()), nil)
	if err != nil {
		return "", nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", nil, c.responseError(res)
	}

	var result struct {
		ID     string           `json:"id"`
		Status ImageJobStatus   `json:"status"`
		Result *embeddingResult `json:"result"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", nil, fmt.Errorf("failed to decode embedding job status: %w", err)
	}

	switch result.Status {
	case ImageJobPending, ImageJobProcessing, ImageJobFailed:
		return result.Status, nil, nil
	case ImageJobCompleted:
		if result.Result == nil || len(result.Result.Vector) != EmbeddingDimension {
			return "", nil, errors.New("embedding job completed without a valid vector")
		}
		return result.Status, result.Result.Vector, nil
	default:
		return "", nil, fmt.Errorf("unknown embedding job status: %s", result.Status)
	}
}

func (c *httpImageSearchClient) Available() bool {
	return c.config.URL != ""
}

func (c *httpImageSearchClient) responseError(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	var e embeddingErrorResponse
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		return fmt.Errorf("embedding server returned %d: %s", res.StatusCode, e.Message)
	}
	return fmt.Errorf("embedding server returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 5 * time.Second
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return 5 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

// nullImageSearchClient 画像検索無効時のnull実装
type nullImageSearchClient struct{}

func (c *nullImageSearchClient) EmbedText(context.Context, string) ([]float64, error) {
	return nil, ErrImageSearchUnavailable
}

func (c *nullImageSearchClient) SubmitImageJob(context.Context, string, string, io.Reader, string) (uuid.UUID, error) {
	return uuid.Nil, ErrImageSearchUnavailable
}

func (c *nullImageSearchClient) GetJob(context.Context, uuid.UUID) (ImageJobStatus, []float64, error) {
	return "", nil, ErrImageSearchUnavailable
}

func (c *nullImageSearchClient) Available() bool {
	return false
}
