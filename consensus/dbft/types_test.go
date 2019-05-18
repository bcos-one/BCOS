package dbft

import (
	"github.com/bcos-one/BCOS/common"
	"math/big"
	"testing"
)

func TestViewCompare(t *testing.T) {
	// test equality
	srvView := &View{
		Sequence: big.NewInt(2),
		Proposer: common.Address{},
	}
	tarView := &View{
		Sequence: big.NewInt(2),
		Proposer: common.Address{},
	}
	if r := srvView.Cmp(tarView); r != 0 {
		t.Errorf("source(%v) should be equal to target(%v): have %v, want %v", srvView, tarView, r, 0)
	}

	// test larger Sequence
	tarView = &View{
		Sequence: big.NewInt(1),
		Proposer: common.Address{},
	}
	if r := srvView.Cmp(tarView); r != 1 {
		t.Errorf("source(%v) should be larger than target(%v): have %v, want %v", srvView, tarView, r, 1)
	}

	// test smaller Sequence
	tarView = &View{
		Sequence: big.NewInt(3),
		Proposer: common.Address{},
	}
	if r := srvView.Cmp(tarView); r != -1 {
		t.Errorf("source(%v) should be smaller than target(%v): have %v, want %v", srvView, tarView, r, -1)
	}
}
