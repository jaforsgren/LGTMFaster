package common

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidIdentifierFormat = errors.New("invalid PR identifier format")
	ErrProviderMismatch        = errors.New("provider type mismatch")
	ErrPartialReviewSubmission = errors.New("review submission partially completed")
)

var messagePattern = regexp.MustCompile(`Message:([^}]+)`)

func ExtractErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	if matches := messagePattern.FindStringSubmatch(errStr); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	if idx := strings.Index(errStr, "Message:"); idx != -1 {
		msg := errStr[idx+8:]
		if endIdx := strings.IndexAny(msg, "}]"); endIdx != -1 {
			msg = msg[:endIdx]
		}
		return strings.TrimSpace(msg)
	}

	if strings.Contains(errStr, "422") || strings.Contains(errStr, "Unprocessable Entity") {
		parts := strings.Split(errStr, ":")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}

	return errStr
}
