package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const numDigits = 6

func GenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate <file name>",
		Short:   "Creates new migrations",
		Example: "morph generate create_users --driver postgresql --dir db/migrations --timestamp",
		Args:    cobra.ExactArgs(1),
		Run:     generateCmdF,
	}

	cmd.Flags().StringP("driver", "d", "", "the driver to use.")
	cmd.Flags().BoolP("timestamp", "t", false, "a timestamp prefix will be added to migration file if set.")
	cmd.Flags().StringP("timeformat", "f", "unix", "timestamp format to be used for timestamps.")
	cmd.Flags().StringP("timezone", "z", "utc", "time zone to be used for timestamps.")
	cmd.Flags().BoolP("sequence", "s", false, "a sequence number prefix will be added to migration file if set.")
	_ = cmd.MarkFlagRequired("driver")

	return cmd
}

func generateCmdF(cmd *cobra.Command, args []string) {
	dir, _ := cmd.Flags().GetString("dir")
	driver, _ := cmd.Flags().GetString("driver")
	extension := getExtension(driver)
	fileName := args[0]

	if ts, _ := cmd.Flags().GetBool("timestamp"); ts {
		tz, _ := cmd.Flags().GetString("timezone")
		loc, err := time.LoadLocation(tz)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		date := time.Now().In(loc)

		tf, _ := cmd.Flags().GetString("timeformat")
		switch tf {
		case "unix":
			fileName = strings.Join([]string{strconv.FormatInt(date.Unix(), 10), fileName}, "_")
		case "unix-nano":
			fileName = strings.Join([]string{strconv.FormatInt(date.UnixNano(), 10), fileName}, "_")
		default:
			fileName = strings.Join([]string{date.Format(tf), fileName}, "_")
		}
	} else if seq, _ := cmd.Flags().GetBool("sequence"); seq {
		next, err := sequelNumber(dir, extension, driver)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}

		version := fmt.Sprintf("%0[2]*[1]d", next, numDigits)
		if len(version) > numDigits {
			cmd.PrintErrf("next sequence number is has %d digit(s), max %d digits allowed.\n", len(version), numDigits)
			return
		}

		fileName = strings.Join([]string{version, fileName}, "_")
	}

	dir = filepath.Join(dir, driver)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		cmd.PrintErrln(err)
		return
	}

	migrations := []string{"down", "up"}
	for _, migration := range migrations {
		filePath := strings.Join([]string{filepath.Join(dir, fileName), migration, extension}, ".")
		f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			cmd.PrintErrln(err)
			return
		}
		f.Close()
	}
}

func sequelNumber(dir, extension, driver string) (int, error) {
	matches, err := filepath.Glob(filepath.Join(dir, driver, "*"+extension))
	if err != nil {
		return 0, err
	}

	if len(matches) == 0 {
		return 1, nil
	}

	filename := matches[len(matches)-1]
	matchSeqStr := filepath.Base(filename)
	idx := strings.Index(matchSeqStr, "_")

	if idx < 0 {
		return 0, fmt.Errorf("invalid migration sequence number: %s", filename)
	}

	matchSeqStr = matchSeqStr[0:idx]
	sequel, err := strconv.ParseInt(matchSeqStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return int(sequel + 1), nil
}

func getExtension(driver string) string {
	switch driver {
	case "postgres", "mysql":
		return "sql"
	default:
		return "txt"
	}
}
