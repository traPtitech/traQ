package search

import (
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestEsMessageMapping_ImageVector(t *testing.T) {
	t.Parallel()

	properties := esMessageMapping()["properties"].(m)

	if _, ok := properties["imageText"]; ok {
		t.Fatal("imageText must not be mapped")
	}

	imageVector, ok := properties["imageVector"].(m)
	if !ok {
		t.Fatal("imageVector must be mapped")
	}
	if imageVector["type"] != "nested" {
		t.Fatalf("unexpected imageVector type: %v", imageVector["type"])
	}

	nested := imageVector["properties"].(m)
	vector := nested["vector"].(m)
	if vector["type"] != "dense_vector" {
		t.Fatalf("unexpected vector type: %v", vector["type"])
	}
	if vector["dims"] != EmbeddingDimension {
		t.Fatalf("unexpected vector dims: %v", vector["dims"])
	}
	if vector["similarity"] != "cosine" {
		t.Fatalf("unexpected vector similarity: %v", vector["similarity"])
	}
	for _, field := range []string{"jobId", "fileId"} {
		if nested[field].(m)["type"] != "keyword" {
			t.Fatalf("%s must be keyword", field)
		}
	}
}

func TestIsEmbeddableImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		meta *model.FileMeta
		want bool
	}{
		{"png", &model.FileMeta{Mime: "image/png", Size: 1}, true},
		{"jpeg", &model.FileMeta{Mime: "image/jpeg", Size: 1}, true},
		{"webp", &model.FileMeta{Mime: "image/webp", Size: 1}, true},
		{"gif", &model.FileMeta{Mime: "image/gif", Size: 1}, false},
		{"animated", &model.FileMeta{Mime: "image/png", Size: 1, IsAnimatedImage: true}, false},
		{"too-large", &model.FileMeta{Mime: "image/png", Size: maxEmbeddingImageSize + 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEmbeddableImage(tt.meta); got != tt.want {
				t.Fatalf("isEmbeddableImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
