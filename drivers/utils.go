package drivers

import (
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
