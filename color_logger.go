package morph

import "github.com/fatih/color"

var (
	ErrorLogger      = color.New(color.FgRed, color.Bold)
	ErrorLoggerLight = color.New(color.FgRed)
	InfoLogger       = color.New(color.FgCyan, color.Bold)
	SuccessLogger    = color.New(color.FgGreen, color.Bold)
)
