package models

import "io"

type Migration struct {
	Bytes   io.ReadCloser
	Name    string
	Version uint32
}
