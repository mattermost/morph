package models

import "io"

type Migration struct {
	bytes io.ReadCloser
	fileName string
}