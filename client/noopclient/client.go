package noopclient

import (
	"context"
	"github.com/brinick/github/client"
)

// ------------------------------------------------------------------

func NewClient() *NoOpClient {
	return &NoOpClient{}
}

// ------------------------------------------------------------------

type NoOpClient struct {
}

func (n *NoOpClient) Get(url string, useStableAPI bool) *client.Page {
	return nil
}
func (n *NoOpClient) GetWithContext(ctx context.Context, url string, useStableAPI bool) *client.Page {
	return nil
}

func (n *NoOpClient) Post(url string, useStableAPI bool, data map[string]string) (int, error) {
	return 0, nil
}
func (n *NoOpClient) PostWithContext(ctx context.Context, url string, useStableAPI bool, data map[string]string) *client.Page {
	return nil
}

func (n *NoOpClient) Patch(url string, useStableAPI bool) *client.Page {
	return nil
}
func (n *NoOpClient) PatchWithContext(ctx context.Context, url string, useStableAPI bool) *client.Page {
	return nil
}
