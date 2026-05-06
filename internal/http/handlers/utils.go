package handlers

import (
	"errors"
	"net/url"
)

var (
	errEmptyURL  error = errors.New("empty url provided")
	errBadScheme error = errors.New("only https, http, and relative scheme can be used")
)

const fallbackURL = "/"

// sanitizeURLToRelative sanitizes backURLs, stripping hostname part of the
// url, as well as restricting scheme to either http, https or relative.
func sanitizeURLToRelative(urlString string) (string, error) {
	if urlString == "" {
		return fallbackURL, errEmptyURL
	}
	backURL, err := url.Parse(urlString)
	if err != nil {
		return fallbackURL, errEmptyURL
	}
	switch backURL.Scheme {
	case "http", "https", "":
	default:
		return fallbackURL, errBadScheme
	}

	backURL.Host = ""
	backURL.Scheme = ""
	backURL.User = nil

	sanitizedString := backURL.String()
	return sanitizedString, nil
}
