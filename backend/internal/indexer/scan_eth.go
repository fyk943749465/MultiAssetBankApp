package indexer

import (
	"context"
	"errors"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	// fallbackConfirmations：节点不支持 finalized/safe 标签（部分 L2、旧节点）时，用 latest 减该值作为扫描上界。
	fallbackConfirmations = uint64(12)
	rpcRetryMax           = 10
	rpcRetryInitial       = 400 * time.Millisecond
	rpcRetryMaxWait       = 45 * time.Second
)

func isRateLimitedRPC(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "429") ||
		strings.Contains(s, "too many requests") ||
		strings.Contains(s, "-32005") ||
		strings.Contains(s, "rate limit") ||
		strings.Contains(s, "exceeded")
}

// ethWithRPCRetry 在 Infura 等返回 429 / -32005 时做指数退避重试；ctx 取消时立即结束。
func ethWithRPCRetry(ctx context.Context, op func() error) error {
	wait := rpcRetryInitial
	var last error
	for attempt := 0; attempt < rpcRetryMax; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = op()
		if last == nil {
			return nil
		}
		if !isRateLimitedRPC(last) {
			return last
		}
		if attempt == rpcRetryMax-1 {
			break
		}
		log.Printf("indexer: RPC rate limited, backing off %v (%v)", wait, last)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		next := wait * 2
		if next > rpcRetryMaxWait {
			next = rpcRetryMaxWait
		}
		wait = next
	}
	return last
}

func ethHeaderByNumber(ctx context.Context, eth *ethclient.Client, num *big.Int) (*types.Header, error) {
	var h *types.Header
	err := ethWithRPCRetry(ctx, func() error {
		var e error
		h, e = eth.HeaderByNumber(ctx, num)
		return e
	})
	return h, err
}

// ethConfirmedTip 返回索引可安全扫到的最高区块号：优先 PoS 的 finalized，其次 safe，最后回退为 latest - fallbackConfirmations。
func ethConfirmedTip(ctx context.Context, eth *ethclient.Client, latest *types.Header) (uint64, error) {
	n, _, err := ethConfirmedTipWithSource(ctx, eth, latest)
	return n, err
}

// ethConfirmedTipWithSource 与 ethConfirmedTip 相同，并返回实际采用的标签（供管理台展示）。
func ethConfirmedTipWithSource(ctx context.Context, eth *ethclient.Client, latest *types.Header) (uint64, string, error) {
	if latest == nil {
		return 0, "", errors.New("indexer: nil latest header")
	}
	latestNum := latest.Number.Uint64()

	try := func(tag int64) (uint64, bool) {
		h, err := ethHeaderByNumber(ctx, eth, big.NewInt(tag))
		if err != nil || h == nil {
			return 0, false
		}
		n := h.Number.Uint64()
		if n > latestNum {
			return 0, false
		}
		return n, true
	}

	if n, ok := try(int64(rpc.FinalizedBlockNumber)); ok {
		return n, "finalized", nil
	}
	if n, ok := try(int64(rpc.SafeBlockNumber)); ok {
		return n, "safe", nil
	}

	if latestNum > fallbackConfirmations {
		return latestNum - fallbackConfirmations, "latest_minus_12", nil
	}
	return 0, "latest_minus_12", nil
}

// ChainRPCHeads 供管理台与游标对照：latest / safe / finalized 及索引器使用的 confirmed 上界。
type ChainRPCHeads struct {
	LatestBlock       uint64  `json:"latest_block"`
	SafeBlock         *uint64 `json:"safe_block,omitempty"`
	FinalizedBlock    *uint64 `json:"finalized_block,omitempty"`
	ConfirmedTipBlock uint64  `json:"confirmed_tip_block"`
	ConfirmedTipSource string `json:"confirmed_tip_source"` // finalized | safe | latest_minus_12
}

// FetchChainRPCHeads 查询当前 RPC 上的链头（与扫块索引用同一套 ethHeaderByNumber 重试）。
func FetchChainRPCHeads(ctx context.Context, eth *ethclient.Client) (ChainRPCHeads, error) {
	var out ChainRPCHeads
	if eth == nil {
		return out, errors.New("indexer: nil eth client")
	}
	latestHdr, err := ethHeaderByNumber(ctx, eth, nil)
	if err != nil {
		return out, err
	}
	out.LatestBlock = latestHdr.Number.Uint64()

	if h, e := ethHeaderByNumber(ctx, eth, big.NewInt(int64(rpc.SafeBlockNumber))); e == nil && h != nil && h.Number != nil {
		n := h.Number.Uint64()
		out.SafeBlock = &n
	}
	if h, e := ethHeaderByNumber(ctx, eth, big.NewInt(int64(rpc.FinalizedBlockNumber))); e == nil && h != nil && h.Number != nil {
		n := h.Number.Uint64()
		out.FinalizedBlock = &n
	}

	tip, src, err := ethConfirmedTipWithSource(ctx, eth, latestHdr)
	if err != nil {
		return out, err
	}
	out.ConfirmedTipBlock = tip
	out.ConfirmedTipSource = src
	return out, nil
}
