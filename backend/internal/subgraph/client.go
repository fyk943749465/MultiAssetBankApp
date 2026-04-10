package subgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client calls a Subgraph Studio (or compatible) GraphQL HTTP endpoint.
type Client struct {
	endpoint string
	apiKey   string
	hc       *http.Client
}

func New(endpoint, apiKey string) *Client {
	endpoint = strings.TrimSpace(endpoint)
	apiKey = strings.TrimSpace(apiKey)
	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		hc:       &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.endpoint != ""
}

type gqlRequest struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Query runs a GraphQL document and returns the raw "data" JSON object.
func (c *Client) Query(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	if !c.Configured() {
		return nil, fmt.Errorf("subgraph: endpoint not configured")
	}
	body, err := json.Marshal(gqlRequest{Query: query, Variables: variables})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("subgraph: HTTP %d: %s", res.StatusCode, truncate(string(raw), 512))
	}
	var out gqlResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("subgraph: decode response: %w", err)
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("subgraph: %s", out.Errors[0].Message)
	}
	return out.Data, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
