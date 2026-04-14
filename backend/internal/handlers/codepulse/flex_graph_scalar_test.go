package codepulse

import (
	"encoding/json"
	"testing"
)

func TestFlexGraphScalarNumberOrString(t *testing.T) {
	var a struct {
		V flexGraphScalar `json:"v"`
	}
	if err := json.Unmarshal([]byte(`{"v":42}`), &a); err != nil {
		t.Fatal(err)
	}
	if string(a.V) != "42" {
		t.Fatalf("got %q", a.V)
	}
	if err := json.Unmarshal([]byte(`{"v":"99"}`), &a); err != nil {
		t.Fatal(err)
	}
	if string(a.V) != "99" {
		t.Fatalf("got %q", a.V)
	}
}
