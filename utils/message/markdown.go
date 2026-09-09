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

type markdownRuntime struct {
	mu        sync.Mutex
	origin    string
	runtime   *markdown.Runtime
	parser    *markdown.Parser
	extractor *markdown.Extractor
	renderer  *markdown.PlainTextRenderer
}

var markdownState = markdownRuntime{origin: "http://localhost:3000"}

// SetOrigin configures same-origin links before serving requests.
func SetOrigin(origin string) {
	markdownState.mu.Lock()
	defer markdownState.mu.Unlock()

	markdownState.origin = origin
	markdownState.closeInstances()
}

func (state *markdownRuntime) initialize(ctx context.Context) error {
	if state.runtime == nil {
		runtime, err := markdown.NewRuntime(ctx, parserWasm)
		if err != nil {
			return err
		}

		state.runtime = runtime
	}

	var err error
	if state.parser == nil {
		state.parser, err = state.runtime.NewParser(ctx, markdown.PresetTraQV1)
		if err != nil {
			return err
		}
	}

	if state.extractor == nil {
		state.extractor, err = state.runtime.NewExtractor(ctx, markdown.ExtractorOptions{Origin: state.origin})
		if err != nil {
			return err
		}
	}

	if state.renderer == nil {
		state.renderer, err = state.runtime.NewPlainTextRenderer(ctx, markdown.RendererOptions{Origin: state.origin})
		if err != nil {
			return err
		}
	}

	return nil
}

func (state *markdownRuntime) closeInstances() {
	if state.parser != nil {
		_ = state.parser.Close(context.Background())
		state.parser = nil
	}

	if state.extractor != nil {
		_ = state.extractor.Close(context.Background())
		state.extractor = nil
	}

	if state.renderer != nil {
		_ = state.renderer.Close(context.Background())
		state.renderer = nil
	}
}

// InitializeMarkdown compiles and checks the embedded parser before startup.
func InitializeMarkdown(ctx context.Context) error {
	markdownState.mu.Lock()
	defer markdownState.mu.Unlock()

	return markdownState.initialize(ctx)
}

// CloseMarkdown releases the compiled module and its instances.
func CloseMarkdown() error {
	markdownState.mu.Lock()
	defer markdownState.mu.Unlock()

	if markdownState.runtime == nil {
		return nil
	}

	err := markdownState.runtime.Close(context.Background())
	markdownState.runtime = nil
	markdownState.parser = nil
	markdownState.extractor = nil
	markdownState.renderer = nil
	return err
}

// Parse lends one parsed Document to the extractor and PlainText renderer.
func Parse(ctx context.Context, text string) (*ParseResult, error) {
	var result *ParseResult
	err := withDocument(ctx, text, func(ctx context.Context, document *markdown.Document) error {
		extraction, err := markdownState.extractor.Extract(ctx, document)
		if err != nil {
			return err
		}

		notification, err := markdownState.renderer.Render(ctx, document)
		if err != nil {
			return err
		}

		result = adaptExtraction(extraction, notification)
		return nil
	})

	return result, err
}

func extractMarkdown(ctx context.Context, text string) (*markdown.Extraction, error) {
	var result *markdown.Extraction
	err := withDocument(ctx, text, func(ctx context.Context, document *markdown.Document) error {
		var err error
		result, err = markdownState.extractor.Extract(ctx, document)
		return err
	})

	return result, err
}

func withDocument(ctx context.Context, text string, consume func(context.Context, *markdown.Document) error) error {
	markdownState.mu.Lock()
	defer markdownState.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := markdownState.initialize(ctx); err != nil {
		return fmt.Errorf("initialize Markdown: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	document, err := markdownState.parser.Parse(ctx, text)
	if err == nil {
		err = consume(ctx, document)
	}
	if err != nil {
		// Cancellation may close an instance. Recreate instances on the next request.
		markdownState.closeInstances()
		return fmt.Errorf("process Markdown: %w", err)
	}

	return nil
}

func adaptExtraction(result *markdown.Extraction, notification string) *ParseResult {
	return &ParseResult{
		Embeddings:       embeddedInfos(result.References.Embeddings),
		PlainText:        result.MessageText,
		notificationText: notification,
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
