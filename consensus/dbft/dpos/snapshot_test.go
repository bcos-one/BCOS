package dpos

import (
	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/consensus/dbft"
	"math/big"
	"testing"
)

var (
	validator1 = common.HexToAddress("0x2831a3d42008a643eaa84d3547f6f77bfaa9a731")
	validator2 = common.HexToAddress("0xb8b6ac2ef7a53cc0dd3f53777fca59cecc63ae4e")
	validator3 = common.HexToAddress("0xbe5563bc5acc3a62130aee01ebb47374cee07096")
	validator4 = common.HexToAddress("0xa73fa67292c0e2bbe61227ad304420355f4b8095")
	validator5 = common.HexToAddress("0x9b21a47e933990f6d921df39ef6f6c15c97421d8")
)

var validator dbft.Validators = dbft.Validators{
	validator1,
	validator2,
	validator3,
	validator4,
	validator5,
}

var loopStartTime uint64 = 1521687594

func TestInturn(t *testing.T) {
	snapshot := fakeSnapshot()

	var (
		headerTime  = new(big.Int).SetUint64(loopStartTime)
		blockNumber = new(big.Int).SetUint64(0)
	)

	if !snapshot.Inturn(validator1, headerTime, blockNumber) {
		t.Fatal("SnapInturn validator1 expecte true actual false")
	}
	if snapshot.Inturn(validator2, headerTime, blockNumber) {
		t.Fatal("SnapInturn validator1 expect false actual true")
	}

	headerTime = new(big.Int).Add(headerTime, big.NewInt(1))
	blockNumber = new(big.Int).Add(blockNumber, big.NewInt(1))
	if !snapshot.Inturn(validator2, headerTime, blockNumber) {
		t.Fatal("SnapInturn validator2 expect true actual false")
	}

	headerTime = new(big.Int).Add(headerTime, big.NewInt(1))
	blockNumber = new(big.Int).Add(blockNumber, big.NewInt(1))
	if !snapshot.Inturn(validator3, headerTime, blockNumber) {
		t.Fatal("SnapInturn validator3 expect true actual false")
	}

	headerTime = new(big.Int).Add(headerTime, big.NewInt(1))
	blockNumber = new(big.Int).Add(blockNumber, big.NewInt(1))
	if !snapshot.Inturn(validator4, headerTime, blockNumber) {
		t.Fatal("SnapInturn validator4 expect true actual false")
	}

	headerTime = new(big.Int).Add(headerTime, big.NewInt(1))
	blockNumber = new(big.Int).Add(blockNumber, big.NewInt(1))
	if !snapshot.Inturn(validator5, headerTime, blockNumber) {
		t.Fatal("SnapInturn validator5 expect true actual false")
	}


	headerTime = new(big.Int).Add(headerTime, big.NewInt(1))
	blockNumber = new(big.Int).Add(blockNumber, big.NewInt(1))
	if !snapshot.Inturn(validator1, headerTime, blockNumber) {
		t.Fatal("SnapInturn validator1 expect true actual false")
	}
}

func TestNextTimeSlot(t *testing.T) {
	snapshot := fakeSnapshot()

	//expectTime := new(big.Int).Add(new(big.Int).SetUint64(loopStartTime), big.NewInt(2))

	next := snapshot.NextTimeSlot(validator3).Uint64()
	if int(next - snapshot.LoopStartTime) % len(validator) != 2 {
		t.Fatal("TestNextTimeSlot fail")
	}
}

func fakeSnapshot() *Snapshot {
	dpos := &DPos{
		config: dbft.DefaultConfig,
	}

	return &Snapshot{
		dpos:          dpos,
		Number:        0,
		Hash:          common.Hash{},
		Validator:     validator,
		Recents:       make(map[uint64]common.Address),
		LoopStartTime: loopStartTime,
	}
}
