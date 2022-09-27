package images

import "embed"

// FS migrateion file system
//
//go:embed *.png
var ImageFS embed.FS
