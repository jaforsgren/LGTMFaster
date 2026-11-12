package domain

import (
	"strings"
	"time"
)

const shortCommitIDLength = 8

// Commit is one commit of a pull request.
type Commit struct {
	ID      string
	Message string
	Author  User
	Date    time.Time
}

func (c Commit) ShortID() string {
	return c.ID[:min(len(c.ID), shortCommitIDLength)]
}

// Subject is the first line of the message.
func (c Commit) Subject() string {
	subject, _, _ := strings.Cut(strings.TrimSpace(c.Message), "\n")
	return subject
}
