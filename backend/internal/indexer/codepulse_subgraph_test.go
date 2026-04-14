package indexer

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSubgraphEntityIDToTxLog(t *testing.T) {
	txh := common.HexToHash("0xcd69f3a7bce8df8a628040b6d8d74d7680d4747f445c0a0634773177d63cafa9")
	li := uint32(7)
	idBytes := append(txh.Bytes(), byte(li>>24), byte(li>>16), byte(li>>8), byte(li))
	id := "0x" + hex.EncodeToString(idBytes)

	gotTx, gotLi, err := subgraphEntityIDToTxLog(id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotTx != txh.Hex() {
		t.Fatalf("tx: got %s want %s", gotTx, txh.Hex())
	}
	if gotLi != int(li) {
		t.Fatalf("logIndex: got %d want %d", gotLi, li)
	}
}
