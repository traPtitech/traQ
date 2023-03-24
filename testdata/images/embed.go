package images

import "embed"

// FS migration file system
//
//go:embed *.png
var ImageFS embed.FS
