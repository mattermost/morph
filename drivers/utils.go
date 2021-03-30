package drivers

import (
	"fmt"
	"hash/crc32"
	"net/url"
)

func ParseCustomParams(conn string, params []string) (map[string]string, error) {
	uri, err := url.Parse(conn)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, param := range params {
		if v := uri.Query().Get(param); v != "" {
			result[param] = v
		}
	}

	return result, nil
}

func SanitizeConnURL(conn string, params []string) (string, error) {
	uri, err := url.Parse(conn)
	if err != nil {
		return conn, err
	}

	query := uri.Query()
	for _, param := range params {
		query.Del(param)
	}
	uri.RawQuery = query.Encode()

	return uri.String(), nil
}

const advisoryLockIDSalt uint = 1486364155

func GenerateAdvisoryLockID(databaseName, schemaName string) (string, error) {
	databaseName = schemaName + databaseName + "\x00"
	sum := crc32.ChecksumIEEE([]byte(databaseName))
	sum = sum * uint32(advisoryLockIDSalt)
	return fmt.Sprint(sum), nil
}
