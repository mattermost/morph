package postgres

import "net/url"

func extractDatabaseName(URL string) (string, error) {
	uri, err := url.Parse(URL)
	if err != nil {
		return "", err
	}

	return uri.Path[1:], nil
}
