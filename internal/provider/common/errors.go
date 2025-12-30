package common

import "errors"

var (
	ErrInvalidIdentifierFormat = errors.New("invalid PR identifier format")
	ErrProviderMismatch        = errors.New("provider type mismatch")
	ErrPartialReviewSubmission = errors.New("review submission partially completed")
)
