package chain

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

// Client wraps go-ethereum RPC client for on-chain operations.
type Client struct {
	eth *ethclient.Client
}

func Dial(rpcURL string) (*Client, error) {
	if rpcURL == "" {
		return nil, nil
	}
	c, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("ethclient dial: %w", err)
	}
	return &Client{eth: c}, nil
}

func (c *Client) Close() {
	if c == nil || c.eth == nil {
		return
	}
	c.eth.Close()
}

// Eth returns the underlying client for advanced use (contracts, filters, etc.).
func (c *Client) Eth() *ethclient.Client {
	if c == nil {
		return nil
	}
	return c.eth
}

// ChainID fetches the chain id from the node (sanity check).
func (c *Client) ChainID(ctx context.Context) (*uint64, error) {
	if c == nil || c.eth == nil {
		return nil, fmt.Errorf("chain client not configured")
	}
	id, err := c.eth.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	v := id.Uint64()
	return &v, nil
}
