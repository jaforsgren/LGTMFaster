package common

import (
	"fmt"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
)

func ParseGitHubIdentifier(identifier string) (owner, repo string, number int, err error) {
	parts := strings.Split(identifier, "/")
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("%w: expected 'owner/repo/number', got '%s'", ErrInvalidIdentifierFormat, identifier)
	}

	owner = parts[0]
	repo = parts[1]
	_, err = fmt.Sscanf(parts[2], "%d", &number)
	if err != nil {
		return "", "", 0, fmt.Errorf("%w: invalid PR number '%s'", ErrInvalidIdentifierFormat, parts[2])
	}

	if owner == "" || repo == "" || number <= 0 {
		return "", "", 0, fmt.Errorf("%w: owner, repo, and number must be non-empty and positive", ErrInvalidIdentifierFormat)
	}

	return owner, repo, number, nil
}

func ParseAzureDevOpsIdentifier(identifier string) (project, repo string, number int, err error) {
	parts := strings.Split(identifier, "/")
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("%w: expected 'project/repo/number', got '%s'", ErrInvalidIdentifierFormat, identifier)
	}

	project = parts[0]
	repo = parts[1]
	_, err = fmt.Sscanf(parts[2], "%d", &number)
	if err != nil {
		return "", "", 0, fmt.Errorf("%w: invalid PR number '%s'", ErrInvalidIdentifierFormat, parts[2])
	}

	if project == "" || repo == "" || number <= 0 {
		return "", "", 0, fmt.Errorf("%w: project, repo, and number must be non-empty and positive", ErrInvalidIdentifierFormat)
	}

	return project, repo, number, nil
}

func FormatPRIdentifier(id domain.PRIdentifier) string {
	return fmt.Sprintf("%s/%d", id.Repository, id.Number)
}
