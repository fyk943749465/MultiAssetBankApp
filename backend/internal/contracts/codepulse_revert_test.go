package contracts

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDecodeRevertErrorTargetBelowMinimumSelector(t *testing.T) {
	abiDef, err := LoadCodePulseABI()
	if err != nil {
		t.Fatal(err)
	}
	cp := &CodePulse{abi: abiDef}
	// TargetBelowMinimum() 选择器（与链上 revert 一致）
	data := common.Hex2Bytes("dcb60468")
	name, _, err := cp.DecodeRevertError(data)
	if err != nil {
		t.Fatal(err)
	}
	if name != "TargetBelowMinimum" {
		t.Fatalf("got error name %q", name)
	}
}
