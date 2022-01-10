package sqlite

import (
	"path/filepath"
	"strings"
)

func extractDatabaseNameFromURL(conn string) string {
	file := filepath.Base(conn)
	file = strings.SplitAfter(file, ".")[0]

	return file
}
