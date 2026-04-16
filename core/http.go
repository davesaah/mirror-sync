package core

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
)

// HTTPClient is a reusable HTTP Client with common actions in making requests for mirror-sync
type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{},
	}
}
func (c *HTTPClient) DoRequest(method, urlStr string, payload []byte, headers map[string]string) (*http.Response, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	req, err := http.NewRequest(method, u.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	return resp, nil
}
