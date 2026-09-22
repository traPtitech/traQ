package message

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	markdown "github.com/traPtitech/traq-flavored-markdown/packages/sdk/go"
)

type markdownRuntime struct {
	mu        sync.RWMutex
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
	markdownState.closeInstancesLocked()
}

func (state *markdownRuntime) initializedLocked() bool {
	return state.runtime != nil && state.parser != nil && state.extractor != nil && state.renderer != nil
}

func (state *markdownRuntime) initializeLocked(ctx context.Context) error {
	if state.runtime == nil {
		runtime, err := markdown.NewBundledRuntime(ctx)
		if err != nil {
			return err
		}

		state.runtime = runtime
	}

	var err error
	if state.parser == nil {
		state.parser, err = state.runtime.NewParser(ctx, markdown.PresetTraqV1)
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

func (state *markdownRuntime) closeInstancesLocked() {
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

	return markdownState.initializeLocked(ctx)
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
	err := markdownState.withDocument(ctx, text, func(ctx context.Context, document *markdown.Document, extractor *markdown.Extractor, renderer *markdown.PlainTextRenderer) error {
		extraction, err := extractor.Extract(ctx, document)
		if err != nil {
			return err
		}

		notification, err := renderer.Render(ctx, document)
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
	err := markdownState.withDocument(ctx, text, func(ctx context.Context, document *markdown.Document, extractor *markdown.Extractor, _ *markdown.PlainTextRenderer) error {
		var err error
		result, err = extractor.Extract(ctx, document)
		return err
	})

	return result, err
}

func (state *markdownRuntime) withDocument(ctx context.Context, text string, consume func(context.Context, *markdown.Document, *markdown.Extractor, *markdown.PlainTextRenderer) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	for {
		state.mu.RLock()
		if !state.initializedLocked() {
			state.mu.RUnlock()

			state.mu.Lock()
			var err error
			if !state.initializedLocked() {
				err = state.initializeLocked(ctx)
			}
			state.mu.Unlock()
			if err != nil {
				return fmt.Errorf("initialize Markdown: %w", err)
			}
			continue
		}

		operationCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		document, err := state.parser.Parse(operationCtx, text)
		if err == nil {
			err = consume(operationCtx, document, state.extractor, state.renderer)
		}
		cancel()
		state.mu.RUnlock()

		if err == nil {
			return nil
		}

		// Cancellation may close an instance. Recreate instances on the next request.
		state.mu.Lock()
		state.closeInstancesLocked()
		state.mu.Unlock()
		return fmt.Errorf("process Markdown: %w", err)
	}
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
