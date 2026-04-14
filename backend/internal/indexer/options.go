package indexer

import (
	"sync"
	"time"
)

var (
	indexerMu sync.RWMutex

	// 以下默认值偏保守，减轻 Infura 等托管节点的 429；可在进程启动时通过 Configure 覆盖。
	pollInterval           = 35 * time.Second
	filterChunkPauseDur    = 800 * time.Millisecond
	maxFilterBlockSpan uint64 = 1000
)

// PollInterval 返回当前索引器轮询周期（Bank、Code Pulse RPC 共用）。
func PollInterval() time.Duration {
	indexerMu.RLock()
	defer indexerMu.RUnlock()
	return pollInterval
}

// Configure 在启动任何索引器 Run 之前调用，用于从环境变量/配置注入降频参数。
// 传入 0 或负数时长、0 跨度表示保持当前默认值不变。
func Configure(poll time.Duration, chunkPause time.Duration, maxBlockSpan uint64) {
	indexerMu.Lock()
	defer indexerMu.Unlock()
	if poll > 0 {
		pollInterval = poll
	}
	if chunkPause > 0 {
		filterChunkPauseDur = chunkPause
	}
	if maxBlockSpan > 0 {
		maxFilterBlockSpan = maxBlockSpan
	}
}

func filterChunkPause() time.Duration {
	indexerMu.RLock()
	defer indexerMu.RUnlock()
	return filterChunkPauseDur
}

func maxBlockSpan() uint64 {
	indexerMu.RLock()
	defer indexerMu.RUnlock()
	return maxFilterBlockSpan
}
