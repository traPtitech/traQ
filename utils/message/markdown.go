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

type markdownProcessor struct {
	mu        sync.Mutex
	origin    string
	runtime   *markdown.Runtime
	processor *markdown.Processor
}

var markdownState = markdownProcessor{origin: "http://localhost:3000"}

// SetOrigin configures same-origin links before serving requests.
func SetOrigin(origin string) {
	markdownState.mu.Lock()
	defer markdownState.mu.Unlock()
	markdownState.origin = origin
	markdownState.closeProcessor()
}

func (state *markdownProcessor) initialize(ctx context.Context) error {
	if state.runtime == nil {
		runtime, err := markdown.NewRuntime(ctx, parserWasm)
		if err != nil {
			return err
		}

		state.runtime = runtime
	}

	if state.processor == nil {
		processor, err := state.runtime.NewProcessor(ctx, markdown.ProcessorPresetTraQV1, markdown.ProcessorOptions{Origin: state.origin})
		if err != nil {
			return err
		}
		state.processor = processor
	}
	return nil
}

func (state *markdownProcessor) closeProcessor() {
	if state.processor == nil {
		return
	}
	_ = state.processor.Close(context.Background())
	state.processor = nil
}

// InitializeMarkdown compiles and checks the embedded parser before startup.
func InitializeMarkdown(ctx context.Context) error {
	markdownState.mu.Lock()
	defer markdownState.mu.Unlock()
	return markdownState.initialize(ctx)
}

// CloseMarkdown releases the compiled module and its instance.
func CloseMarkdown() error {
	markdownState.mu.Lock()
	defer markdownState.mu.Unlock()
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
	markdownState.mu.Lock()
	defer markdownState.mu.Unlock()

	// Reject cancelled requests before touching the Wasm runtime.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Lazily initialize the runtime for callers that do not use the CLI.
	if err := markdownState.initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize Markdown: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// The processor instance is serialized by markdownState.mu.
	result, err := markdownState.processor.Process(ctx, text)
	if err != nil {
		// Cancellation closes a Wasm instance. The next request gets a fresh instance.
		markdownState.closeProcessor()
		return nil, fmt.Errorf("process Markdown: %w", err)
	}

	// Keep the Rust SDK details inside this package's compatibility boundary.
	return adaptProcessOutput(result), nil
}

func adaptProcessOutput(result *markdown.ProcessOutput) *ParseResult {
	return &ParseResult{
		Embeddings:       embeddedInfos(result.References.Embeddings),
		PlainText:        result.PlainText,
		notificationText: result.NotificationText,
		Mentions:         referenceIDs(result.References.Mentions),
		GroupMentions:    referenceIDs(result.References.GroupMentions),
		ChannelLink:      referenceIDs(result.References.ChannelLinks),
		Attachments:      referenceIDs(result.Attachments),
		Citation:         referenceIDs(result.Citations),
	}
}

func embeddedInfos(values []markdown.EmbeddedInfo) []*EmbeddedInfo {
	infos := make([]*EmbeddedInfo, len(values))
	for i := range values {
		infos[i] = &values[i]
	}
	return infos
}

func referenceIDs(values []string) []uuid.UUID {
	if len(values) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(values))
	for i, value := range values {
		ids[i] = uuid.Must(uuid.FromString(value))
	}
	return ids
}
