package testlib

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func ExecuteCommand(t *testing.T, c *cobra.Command, args ...string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	c.SetOut(buf)
	c.SetErr(buf)
	c.SetArgs(args)

	err := c.Execute()
	return strings.TrimSpace(buf.String()), err
}
