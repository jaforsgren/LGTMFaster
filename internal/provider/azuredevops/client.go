package azuredevops

import (
	"context"
	"fmt"
)

type Client struct {
	token        string
	organization string
	username     string
}

func NewClient(token string, organization string, username string) *Client {
	return &Client{
		token:        token,
		organization: organization,
		username:     username,
	}
}

func (c *Client) ValidateCredentials(ctx context.Context) error {
	return fmt.Errorf("azure devops provider not yet implemented")
}
