package message

import (
	"context"
	_ "embed"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	markdown "github.com/traq-markdown-parser/traq/go"
)

//go:embed parser.wasm
var parserWasm []byte

var markdownState = struct {
	sync.Mutex
	origin    string
	runtime   *markdown.Runtime
	processor *markdown.Processor
}{origin: "http://localhost:3000"}

// SetOrigin configures same-origin links before serving requests.
func SetOrigin(origin string) {
	markdownState.Lock()
	defer markdownState.Unlock()
	markdownState.origin = origin
	setMetadataOrigin(origin)
	if markdownState.processor != nil {
		_ = markdownState.processor.Close(context.Background())
		markdownState.processor = nil
	}
}

func initializeMarkdown(ctx context.Context) error {
	if markdownState.runtime == nil {
		runtime, err := markdown.NewRuntime(ctx, parserWasm)
		if err != nil {
			return err
		}
		markdownState.runtime = runtime
	}
	if markdownState.processor == nil {
		processor, err := markdownState.runtime.NewProcessor(ctx, markdown.ProcessorPresetTraQV1, markdown.ProcessorOptions{Origin: markdownState.origin})
		if err != nil {
			return err
		}
		markdownState.processor = processor
	}
	return nil
}

// InitializeMarkdown compiles and checks the embedded parser before startup.
func InitializeMarkdown(ctx context.Context) error {
	markdownState.Lock()
	defer markdownState.Unlock()
	return initializeMarkdown(ctx)
}

// CloseMarkdown releases the compiled module and its instance.
func CloseMarkdown() error {
	markdownState.Lock()
	defer markdownState.Unlock()
	if markdownState.runtime == nil {
		return nil
	}
	err := markdownState.runtime.Close(context.Background())
	markdownState.runtime = nil
	markdownState.processor = nil
	return err
}

// Parse processes Markdown in Rust once. Errors must reach callers before storage.
func Parse(ctx context.Context, text string) (*ParseResult, error) {
	markdownState.Lock()
	defer markdownState.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := initializeMarkdown(ctx); err != nil {
		return nil, fmt.Errorf("initialize Markdown: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result, err := markdownState.processor.Process(ctx, text)
	if err != nil {
		// Cancellation closes a Wasm instance. The next request gets a fresh instance.
		_ = markdownState.processor.Close(context.Background())
		markdownState.processor = nil
		return nil, fmt.Errorf("process Markdown: %w", err)
	}
	parsed := parseMetadata(text)
	parsed.notificationText = result.NotificationText
	parsed.Mentions = referenceIDs(result.References.Mentions)
	parsed.GroupMentions = referenceIDs(result.References.GroupMentions)
	parsed.ChannelLink = referenceIDs(result.References.ChannelLinks)
	return parsed, nil
}

func referenceIDs(values []string) []uuid.UUID {
	var ids []uuid.UUID
	for _, value := range values {
		ids = append(ids, uuid.Must(uuid.FromString(value)))
	}
	return ids
}
