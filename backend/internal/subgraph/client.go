package subgraph

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// ClientConfig tunes optional behavior. Zero value is valid (no response cache).
type ClientConfig struct {
	// QueryCacheTTL: identical GraphQL requests reuse the last response for this duration (reduces Studio query count). 0 disables.
	QueryCacheTTL time.Duration
	// QueryCacheMaxEntries: max distinct cached keys when TTL > 0. If <= 0, defaults to 512.
	QueryCacheMaxEntries int
}

// Client calls a Subgraph Studio (or compatible) GraphQL HTTP endpoint.
type Client struct {
	endpoint string
	apiKey   string
	hc       *http.Client

	cacheTTL time.Duration
	cacheMax int
	cacheMu  sync.Mutex
	cache    map[string]cacheEntry
	sf       singleflight.Group
}

type cacheEntry struct {
	data      json.RawMessage
	expiresAt time.Time
}

// New builds a client. Pass ClientConfig{QueryCacheTTL: 30*time.Second} (for example) to enable caching and in-flight deduplication.
func New(endpoint, apiKey string, cfg ClientConfig) *Client {
	endpoint = strings.TrimSpace(endpoint)
	apiKey = strings.TrimSpace(apiKey)
	max := cfg.QueryCacheMaxEntries
	if max <= 0 {
		max = 512
	}
	cl := &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		hc:       &http.Client{Timeout: 45 * time.Second},
		cacheTTL: cfg.QueryCacheTTL,
		cacheMax: max,
	}
	if cfg.QueryCacheTTL > 0 {
		cl.cache = make(map[string]cacheEntry)
	}
	return cl
}

func (c *Client) Configured() bool {
	return c != nil && c.endpoint != ""
}

// endpointForLog returns host+path for logs (no query string, no API key).
func (c *Client) endpointForLog() string {
	if c == nil {
		return "(nil client)"
	}
	if c.endpoint == "" {
		return "(none)"
	}
	u, err := url.Parse(c.endpoint)
	if err != nil || u.Host == "" {
		return "(unparseable)"
	}
	s := u.Host + u.Path
	if s == "" {
		return u.Host
	}
	return strings.TrimSuffix(s, "/")
}

func (c *Client) logQueryFailure(phase string, err error) {
	if err == nil {
		return
	}
	var ep string
	if c != nil {
		ep = c.endpointForLog()
	} else {
		ep = "(nil client)"
	}
	log.Printf("subgraph query failed [%s] endpoint=%s: %v", phase, ep, err)
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

func copyJSONRawMessage(b json.RawMessage) json.RawMessage {
	if b == nil {
		return nil
	}
	out := make([]byte, len(b))
	copy(out, b)
	return json.RawMessage(out)
}

func (c *Client) cacheKey(query string, variables map[string]any) string {
	var vb []byte
	var err error
	if variables == nil {
		vb = []byte("null")
	} else {
		vb, err = json.Marshal(variables)
		if err != nil {
			vb = []byte(`"marshal_err"`)
		}
	}
	h := sha256.Sum256(append(append([]byte(query), 0), vb...))
	return hex.EncodeToString(h[:])
}

func (c *Client) cacheGet(key string) (json.RawMessage, bool) {
	if c == nil || c.cache == nil {
		return nil, false
	}
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	ent, ok := c.cache[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(ent.expiresAt) {
		delete(c.cache, key)
		return nil, false
	}
	return copyJSONRawMessage(ent.data), true
}

func (c *Client) evictExpiredLocked() {
	if c.cache == nil {
		return
	}
	now := time.Now()
	for k, ent := range c.cache {
		if now.After(ent.expiresAt) {
			delete(c.cache, k)
		}
	}
}

func (c *Client) cachePut(key string, data json.RawMessage) {
	if c == nil || c.cache == nil || c.cacheTTL <= 0 {
		return
	}
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.evictExpiredLocked()
	for len(c.cache) >= c.cacheMax {
		for k := range c.cache {
			delete(c.cache, k)
			break
		}
	}
	c.cache[key] = cacheEntry{
		data:      copyJSONRawMessage(data),
		expiresAt: time.Now().Add(c.cacheTTL),
	}
}

// Query runs a GraphQL document and returns the raw "data" JSON object.
func (c *Client) Query(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	if !c.Configured() {
		err := fmt.Errorf("subgraph: endpoint not configured")
		c.logQueryFailure("not_configured", err)
		return nil, err
	}
	if c.cacheTTL <= 0 || c.cache == nil {
		return c.doQuery(ctx, query, variables)
	}

	key := c.cacheKey(query, variables)
	if data, ok := c.cacheGet(key); ok {
		return data, nil
	}

	v, err, _ := c.sf.Do(key, func() (interface{}, error) {
		raw, err := c.doQuery(ctx, query, variables)
		if err != nil {
			return nil, err
		}
		c.cachePut(key, raw)
		return copyJSONRawMessage(raw), nil
	})
	if err != nil {
		return nil, err
	}
	return copyJSONRawMessage(v.(json.RawMessage)), nil
}

func (c *Client) doQuery(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	body, err := json.Marshal(gqlRequest{Query: query, Variables: variables})
	if err != nil {
		c.logQueryFailure("marshal_request", err)
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		c.logQueryFailure("new_request", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	res, err := c.hc.Do(req)
	if err != nil {
		c.logQueryFailure("http_do", err)
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		c.logQueryFailure("read_body", err)
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		err := fmt.Errorf("subgraph: HTTP %d: %s", res.StatusCode, truncate(string(raw), 512))
		c.logQueryFailure("http_status", err)
		return nil, err
	}
	var out gqlResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		err := fmt.Errorf("subgraph: decode response: %w", err)
		c.logQueryFailure("decode_json", err)
		return nil, err
	}
	if len(out.Errors) > 0 {
		err := fmt.Errorf("subgraph: %s", out.Errors[0].Message)
		c.logQueryFailure("graphql_errors", err)
		return nil, err
	}
	return out.Data, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
