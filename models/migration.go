package models

import "io"

type Migration struct {
	Bytes    io.ReadCloser
	FileName string
}
