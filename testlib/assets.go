package testlib

import "embed"

//go:embed scripts
var assets embed.FS

func Assets() embed.FS {
	return assets
}
