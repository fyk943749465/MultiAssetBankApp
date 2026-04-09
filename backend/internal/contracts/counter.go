package contracts

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

//go:embed counter.json
var counterABIJSON []byte

// Counter wraps the bound contract for get/count.
type Counter struct {
	bound *bind.BoundContract
}

func NewCounter(client *ethclient.Client, address common.Address) (*Counter, error) {
	parsed, err := abi.JSON(bytes.NewReader(counterABIJSON))
	if err != nil {
		return nil, err
	}
	b := bind.NewBoundContract(address, parsed, client, client, client)
	return &Counter{bound: b}, nil
}

func (c *Counter) Get(ctx context.Context) (*big.Int, error) {
	if c == nil || c.bound == nil {
		return nil, fmt.Errorf("counter not initialized")
	}
	var out []interface{}
	opts := &bind.CallOpts{Context: ctx}
	if err := c.bound.Call(opts, &out, "get"); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("get: empty result")
	}
	// proto 必须是 *big.Int 形态：用 new(big.Int)，不要用 new(*big.Int)
	if v, ok := out[0].(*big.Int); ok && v != nil {
		return new(big.Int).Set(v), nil
	}
	cv := abi.ConvertType(out[0], new(big.Int))
	v, ok := cv.(*big.Int)
	if !ok || v == nil {
		return nil, fmt.Errorf("get: unexpected type %T", out[0])
	}
	return v, nil
}

func (c *Counter) Count(ctx context.Context, auth *bind.TransactOpts) (*types.Transaction, error) {
	if c == nil || c.bound == nil {
		return nil, fmt.Errorf("counter not initialized")
	}
	auth.Context = ctx
	return c.bound.Transact(auth, "count")
}
