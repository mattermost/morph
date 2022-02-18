package commands

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-morph/morph/commands/testlib"
	"github.com/stretchr/testify/require"
)

func TestGenerateCMD(t *testing.T) {
	dir := "./tmp"
	cmd := GenerateCmd()
	cmd.PersistentFlags().String("dir", ".", "the migrations directory")

	t.Run("should generate migration files correctly when using --sequence", func(t *testing.T) {
		name := "create_saiyans"

		defer func() {
			err := os.RemoveAll(dir)
			if err != nil {
				log.Fatal(err)
			}
		}()

		// ensure that directory doesn't exist
		_, err := os.Stat("./tmp")
		require.Equal(t, errors.Is(err, os.ErrNotExist), true)

		cases := []struct {
			driver   string
			args     []string
			sequence string
		}{
			{
				driver:   "postgres",
				args:     []string{name, "--dir", dir, "--sequence"},
				sequence: "000001",
			},
			{
				driver:   "mysql",
				args:     []string{name, "--dir", dir, "--sequence"},
				sequence: "000001",
			},
			{
				driver:   "postgres",
				args:     []string{name, "--dir", dir, "--sequence"},
				sequence: "000002",
			},
			{
				driver:   "mysql",
				args:     []string{name, "--dir", dir, "--sequence"},
				sequence: "000002",
			},
			{
				driver:   "postgres",
				args:     []string{name, "--dir", dir, "-s"},
				sequence: "000003",
			},
			{
				driver:   "mysql",
				args:     []string{name, "--dir", dir, "-s"},
				sequence: "000003",
			},
		}

		for _, tc := range cases {
			args := append(tc.args, "--driver", tc.driver)

			_, err := testlib.ExecuteCommand(t, cmd, args...)
			require.NoError(t, err)

			_, fErr := os.Stat(filepath.Join("./tmp", tc.driver, tc.sequence+"_"+name+".down.sql"))
			require.NoError(t, fErr)

			_, fErr = os.Stat(filepath.Join("./tmp/", tc.driver, tc.sequence+"_"+name+".up.sql"))
			require.NoError(t, fErr)
		}
	})

	t.Run("should correctly return extension", func(t *testing.T) {
		ext := getExtension("postgres")
		require.Equal(t, ext, "sql")

		ext = getExtension("mysql")
		require.Equal(t, ext, "sql")

		ext = getExtension("postgresql")
		require.Equal(t, ext, "txt")

		ext = getExtension("mysqlite")
		require.Equal(t, ext, "txt")
	})
}
