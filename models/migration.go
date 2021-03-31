package models

import (
	"bytes"
	"io"
)

type Migration struct {
	Bytes   io.ReadCloser
	Name    string
	Version uint32
}

func (m *Migration) Query() (string, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(m.Bytes); err != nil {
		return "", err
	}
	return buf.String(), nil
}
