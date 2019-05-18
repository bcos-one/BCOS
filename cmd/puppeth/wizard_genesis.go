// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/bcos-one/BCOS/accounts/abi"
	"github.com/bcos-one/BCOS/core/types"
	"github.com/bcos-one/BCOS/crypto/sha3"
	"github.com/bcos-one/BCOS/rlp"
	"io/ioutil"
	"math/big"
	"math/rand"
	"time"

	"github.com/bcos-one/BCOS/common"
	"github.com/bcos-one/BCOS/core"
	"github.com/bcos-one/BCOS/log"
	"github.com/bcos-one/BCOS/params"
)

// dbft vote contract runtime-code
var voteContractCode = "60806040526004361061008d5763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166308ac525681146100925780631cc59aa3146100b957806351cff8d9146100f95780636dd7d8ea1461012e5780638ab66a9014610144578063b7ab4db514610165578063d51b9e93146101ca578063ed612f8c146101eb575b600080fd5b34801561009e57600080fd5b506100a7610200565b60408051918252519081900360200190f35b3480156100c557600080fd5b506100e0600160a060020a0360043581169060243516610206565b6040805192835260208301919091528051918290030190f35b34801561010557600080fd5b5061011a600160a060020a036004351661022a565b604080519115158252519081900360200190f35b610142600160a060020a0360043516610389565b005b34801561015057600080fd5b506100a7600160a060020a0360043516610477565b34801561017157600080fd5b5061017a610489565b60408051602080825283518183015283519192839290830191858101910280838360005b838110156101b657818101518382015260200161019e565b505050509050019250505060405180910390f35b3480156101d657600080fd5b5061011a600160a060020a0360043516610559565b3480156101f757600080fd5b506100a7610575565b60005481565b60046020908152600092835260408084209091529082529020805460019091015482565b336000908152600460209081526040808320600160a060020a03851684529091528120548190811061025b57600080fd5b336000908152600460209081526040808320600160a060020a0387168452909152902060010154620f424001431161029257600080fd5b50336000818152600460209081526040808320600160a060020a03871684529091528082208054838255600190910183905590519092916108fc841502918491818181858888f193505050501580156102ef573d6000803e3d6000fd5b50600160a060020a038316600090815260016020526040902054610319908263ffffffff6105c816565b600160a060020a03841660009081526001602052604090205561033b836105da565b60408051338152600160a060020a038516602082015280820183905290517f9b1bfa7fa9ee420a16e124f794c35ac9f90472acc99140eb2f6447c714cad8eb9181900360600190a150919050565b69d3c21bcecceda10000003410156103a057600080fd5b6103a981610559565b156103fe57600160a060020a0381166000908152600160205260409020546103d7903463ffffffff61073216565b600160a060020a0382166000908152600160205260409020556103f981610748565b610422565b600160a060020a03811660009081526001602052604090203490556104228161085b565b61042d33823461093c565b60408051338152600160a060020a0383166020820152348183015290517f66a9138482c99e9baf08860110ef332cc0c23b4a199a53593d8db0fc8f96fbfc9181900360600190a150565b60016020526000908152604090205481565b606080600080610497610575565b6040519080825280602002602001820160405280156104c0578160200160208202803883390190505b5060035490935060009250600160a060020a031690505b600160a060020a0381161561054f578083838151811015156104f557fe5b600160a060020a03909216602092830290910190910152600054600190920191821061052357829350610553565b600160a060020a03908116600090815260026020908152604080832060018452909152902054166104d7565b8293505b50505090565b600160a060020a03166000908152600160205260408120541190565b600354600090600160a060020a03165b600160a060020a038116156105c457600160a060020a039081166000908152600260209081526040808320600180855292529091205492019116610585565b5090565b6000828211156105d457fe5b50900390565b600160a060020a038082166000908152600260209081526040808320600184529091529020541680158061062e5750600160a060020a03808316600090815260016020526040808220549284168252902054105b156106385761072e565b610641826109a9565b600160a060020a03821660009081526001602052604090205415156106655761072e565b50600160a060020a03808216600090815260026020908152604080832060018452909152902054165b600160a060020a0380831660009081526001602052604080822054928416825290205410156106c6576106c18282610a93565b61072e565b600160a060020a038181166000908152600260209081526040808320600184529091529020541615156106f857610724565b600160a060020a039081166000908152600260209081526040808320600184529091529020541661068e565b61072e8282610b0c565b5050565b60008282018381101561074157fe5b9392505050565b600160a060020a0380821660009081526002602090815260408083208380529091529020541680158061079c5750600160a060020a0380831660009081526001602052604080822054928416825290205410155b156107a65761072e565b6107af826109a9565b50600160a060020a038082166000908152600260209081526040808320838052909152902054165b600160a060020a0381161561084457600160a060020a038083166000908152600160205260408082205492841682529020541115610819576106c18282610b0c565b600160a060020a039081166000908152600260209081526040808320838052909152902054166107d7565b60035461072e908390600160a060020a0316610a93565b600354600090600160a060020a0316151561089d576003805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a03841617905561072e565b50600354600160a060020a03165b600160a060020a0380821660009081526001602052604080822054928516825290205411156108de576106c18282610a93565b600160a060020a0381811660009081526002602090815260408083206001845290915290205416151561091057610724565b600160a060020a03908116600090815260026020908152604080832060018452909152902054166108ab565b600160a060020a03808416600090815260046020908152604080832093861683529290522054610972908263ffffffff61073216565b600160a060020a03938416600090815260046020908152604080832095909616825293909352929091209182555043600190910155565b600354600160a060020a0382811691161415610a0857600160a060020a038082166000908152600260209081526040808320600184529091529020546003805473ffffffffffffffffffffffffffffffffffffffff1916919092161790555b600160a060020a0381811660009081526002602090815260408083208380529091528082205460018352912054610a43929182169116610b44565b600160a060020a03166000908152600260209081526040808320838052909152808220805473ffffffffffffffffffffffffffffffffffffffff1990811690915560018352912080549091169055565b600160a060020a038082166000908152600260209081526040808320838052909152902054610ac3911683610b44565b600354600160a060020a0382811691161415610b02576003805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0384161790555b61072e8282610b44565b600160a060020a03808216600090815260026020908152604080832060018452909152902054610b3e91849116610b44565b61072e81835b600160a060020a03821615610b9b57600160a060020a038281166000908152600260209081526040808320600184529091529020805473ffffffffffffffffffffffffffffffffffffffff19169183169190911790555b600160a060020a0381161561072e57600160a060020a0390811660009081526002602090815260408083208380529091529020805473ffffffffffffffffffffffffffffffffffffffff1916929091169190911790555600a165627a7a723058204b10c79d0713152cb9f000083d71a1873231f249c04dff8c5b85ea7c1aac93850029"

// makeGenesis creates a new genesis struct based on some user input.
func (w *wizard) makeGenesis() {
	// Construct a default genesis block
	genesis := &core.Genesis{
		Timestamp:  uint64(time.Now().Unix()),
		GasLimit:   4700000,
		Difficulty: big.NewInt(524288),
		Alloc:      make(core.GenesisAlloc),
		Config: &params.ChainConfig{
			HomesteadBlock: big.NewInt(1),
			EIP150Block:    big.NewInt(2),
			EIP155Block:    big.NewInt(3),
			EIP158Block:    big.NewInt(3),
			ByzantiumBlock: big.NewInt(4),
		},
	}
	// Figure out which consensus engine to choose
	fmt.Println()
	fmt.Println("Which consensus engine to use? (default = clique)")
	fmt.Println(" 1. Ethash - proof-of-work")
	fmt.Println(" 2. Clique - proof-of-authority")
	fmt.Println(" 3. DBFT   - DPOS-PBFT")
	fmt.Println(" 4. Bcos - istanbul")
	fmt.Println(" 5. Bcos - proof-of-authority")

	choice := w.read()
	switch {
	case choice == "1":
		// In case of ethash, we're pretty much done
		genesis.Config.Ethash = new(params.EthashConfig)
		genesis.ExtraData = make([]byte, 32)

	case choice == "" || choice == "2":
		// In the case of clique, configure the consensus parameters
		genesis.Difficulty = big.NewInt(1)
		genesis.Config.Clique = &params.CliqueConfig{
			Period: 15,
			Epoch:  30000,
		}
		fmt.Println()
		fmt.Println("How many seconds should blocks take? (default = 15)")
		genesis.Config.Clique.Period = uint64(w.readDefaultInt(15))

		// We also need the initial list of signers
		fmt.Println()
		fmt.Println("Which accounts are allowed to seal? (mandatory at least one)")

		var signers []common.Address
		for {
			if address := w.readAddress(); address != nil {
				signers = append(signers, *address)
				continue
			}
			if len(signers) > 0 {
				break
			}
		}
		// Sort the signers and embed into the extra-data section
		for i := 0; i < len(signers); i++ {
			for j := i + 1; j < len(signers); j++ {
				if bytes.Compare(signers[i][:], signers[j][:]) > 0 {
					signers[i], signers[j] = signers[j], signers[i]
				}
			}
		}
		genesis.ExtraData = make([]byte, 32+len(signers)*common.AddressLength+65)
		for i, signer := range signers {
			copy(genesis.ExtraData[32+i*common.AddressLength:], signer[:])
		}
	case choice == "3":
		w.makeDBFTGenesis()
		return

	case choice == "4":
		genesis.Difficulty = big.NewInt(1)
		genesis.Config.Istanbul = &params.IstanbulConfig{
			ProposerPolicy: 0,
			Epoch:          30000,
			Period:         0,
		}
		genesis.Config.IsBcos = true;
		genesis.Mixhash = types.IstanbulDigest
		w.makeGenesisExtraData(genesis, choice)

	case choice == "5":
		// In the case of clique, configure the consensus parameters
		genesis.Difficulty = big.NewInt(1)
		genesis.Config.WPoa = &params.WPoaConfig{
			Period: 15,
			Epoch:  30000,
		}
		fmt.Println()
		fmt.Println("How many seconds should blocks take? (default = 15)")
		genesis.Config.WPoa.Period = uint64(w.readDefaultInt(15))
		genesis.Config.IsBcos = true;
		w.makeGenesisExtraData(genesis, choice)

	default:
		log.Crit("Invalid consensus engine choice", "choice", choice)
	}
	// Consensus all set, just ask for initial funds and go
	fmt.Println()
	fmt.Println("Which accounts should be pre-funded? (advisable at least one)")
	for {
		// Read the address of the account to fund
		if address := w.readAddress(); address != nil {
			genesis.Alloc[*address] = core.GenesisAccount{
				Balance: new(big.Int).Lsh(big.NewInt(1), 256-7), // 2^256 / 128 (allow many pre-funds without balance overflows)
			}
			continue
		}
		break
	}
	// Add a batch of precompile balances to avoid them getting deleted
	//for i := int64(0); i < 256; i++ {
	//	genesis.Alloc[common.BigToAddress(big.NewInt(i))] = core.GenesisAccount{Balance: big.NewInt(1)}
	//}
	// Query the user for some custom extras
	fmt.Println()
	fmt.Println("Specify your chain/network ID if you want an explicit one (default = random)")
	genesis.Config.ChainID = new(big.Int).SetUint64(uint64(w.readDefaultInt(rand.Intn(65536))))

	// All done, store the genesis and flush to disk
	log.Info("Configured new genesis block")

	w.conf.Genesis = genesis
	w.conf.flush()
}

func (w *wizard) makeGenesisExtraData(genesis *core.Genesis, choice string) {
	var signers   []common.Address
	var managers  []common.Address

	if choice == "5" {
		fmt.Println()
		fmt.Println("Which accounts are should be manager? (mandatory at least one)")
		for {
			if address := w.readAddress(); address != nil {
				managers = append(managers, *address)
				continue
			}
			if len(managers) > 0 {
				break
			}
		}

		// Sort the managers and embed into the extra-data section
		for i := 0; i < len(managers); i++ {
			for j := i + 1; j < len(managers); j++ {
				if bytes.Compare(managers[i][:], managers[j][:]) > 0 {
					managers[i], managers[j] = managers[j], managers[i]
				}
			}
		}
	}


	fmt.Println()
	fmt.Println("Which accounts are allowed to seal? (mandatory at least one)")

	for {
		if address := w.readAddress(); address != nil {
			signers = append(signers, *address)
			continue
		}
		if len(signers) > 0 {
			break
		}
	}

	// Sort the signers and embed into the extra-data section
	for i := 0; i < len(signers); i++ {
		for j := i + 1; j < len(signers); j++ {
			if bytes.Compare(signers[i][:], signers[j][:]) > 0 {
				signers[i], signers[j] = signers[j], signers[i]
			}
		}
	}

	switch {
	case choice == "5":    // wpoa
		wpoa := &types.WPoaExtra{
			Managers: managers,
			Signers:  signers,
		}

		payload, err := rlp.EncodeToBytes(&wpoa)
		if err != nil {
			log.Crit("encode wpoa extradata error")
			break
		}

		genesis.ExtraData = make([]byte, types.WPoaExtraVanity+len(payload)+types.WPoaExtraSeal)
		copy(genesis.ExtraData[types.WPoaExtraVanity:], payload)

	case choice == "4":  // istanbul
		// 生成Istanbul的extraData
		ist := &types.IstanbulExtra{
			Validators:    signers,
			Seal:          []byte{},
			CommittedSeal: [][]byte{},
		}

		payload, err := rlp.EncodeToBytes(&ist)
		if err != nil {
			log.Crit("encode istanbul extradata error")
			break
		}

		genesis.ExtraData = make([]byte, types.IstanbulExtraVanity+len(payload))
		copy(genesis.ExtraData[types.IstanbulExtraVanity:], payload)

	case choice == "2":  // clique
		genesis.ExtraData = make([]byte, 32+len(signers)*common.AddressLength+65)
		for i, signer := range signers {
			copy(genesis.ExtraData[32+i*common.AddressLength:], signer[:])
		}
	default:
		break
	}
}

// manageGenesis permits the modification of chain configuration parameters in
// a genesis config and the export of the entire genesis spec.
func (w *wizard) manageGenesis() {
	// Figure out whether to modify or export the genesis
	fmt.Println()
	fmt.Println(" 1. Modify existing fork rules")
	fmt.Println(" 2. Export genesis configuration")
	fmt.Println(" 3. Remove genesis configuration")

	choice := w.read()
	switch {
	case choice == "1":
		// Fork rule updating requested, iterate over each fork
		fmt.Println()
		fmt.Printf("Which block should Homestead come into effect? (default = %v)\n", w.conf.Genesis.Config.HomesteadBlock)
		w.conf.Genesis.Config.HomesteadBlock = w.readDefaultBigInt(w.conf.Genesis.Config.HomesteadBlock)

		fmt.Println()
		fmt.Printf("Which block should EIP150 come into effect? (default = %v)\n", w.conf.Genesis.Config.EIP150Block)
		w.conf.Genesis.Config.EIP150Block = w.readDefaultBigInt(w.conf.Genesis.Config.EIP150Block)

		fmt.Println()
		fmt.Printf("Which block should EIP155 come into effect? (default = %v)\n", w.conf.Genesis.Config.EIP155Block)
		w.conf.Genesis.Config.EIP155Block = w.readDefaultBigInt(w.conf.Genesis.Config.EIP155Block)

		fmt.Println()
		fmt.Printf("Which block should EIP158 come into effect? (default = %v)\n", w.conf.Genesis.Config.EIP158Block)
		w.conf.Genesis.Config.EIP158Block = w.readDefaultBigInt(w.conf.Genesis.Config.EIP158Block)

		fmt.Println()
		fmt.Printf("Which block should Byzantium come into effect? (default = %v)\n", w.conf.Genesis.Config.ByzantiumBlock)
		w.conf.Genesis.Config.ByzantiumBlock = w.readDefaultBigInt(w.conf.Genesis.Config.ByzantiumBlock)

		out, _ := json.MarshalIndent(w.conf.Genesis.Config, "", "  ")
		fmt.Printf("Chain configuration updated:\n\n%s\n", out)

	case choice == "2":
		// Save whatever genesis configuration we currently have
		fmt.Println()
		fmt.Printf("Which file to save the genesis into? (default = %s.json)\n", w.network)
		out, _ := json.MarshalIndent(w.conf.Genesis, "", "  ")
		if err := ioutil.WriteFile(w.readDefaultString(fmt.Sprintf("%s.json", w.network)), out, 0644); err != nil {
			log.Error("Failed to save genesis file", "err", err)
		}
		log.Info("Exported existing genesis block")

	case choice == "3":
		// Make sure we don't have any services running
		if len(w.conf.servers()) > 0 {
			log.Error("Genesis reset requires all services and servers torn down")
			return
		}
		log.Info("Genesis block destroyed")

		w.conf.Genesis = nil
		w.conf.flush()

	default:
		log.Error("That's not something I can do")
	}
}

func (w *wizard) makeDBFTGenesis() {
	genesis := &core.Genesis{
		Timestamp:  uint64(time.Now().Unix()),
		GasLimit:   0x7A1200,
		Difficulty: big.NewInt(1),
		Alloc:      make(core.GenesisAlloc),
		Config: &params.ChainConfig{
			HomesteadBlock: big.NewInt(1),
			EIP150Block:    big.NewInt(2),
			EIP155Block:    big.NewInt(3),
			EIP158Block:    big.NewInt(3),
			ByzantiumBlock: big.NewInt(4),
			ChainID:        big.NewInt(2019),
			Dbft: &params.DbftConfig{
				Epoch:       30000,
				BlockPeriod: 1,
			},
		},
		ExtraData: make([]byte, 32),
	}

	fmt.Println()
	fmt.Println("How many seconds should blocks take? (default = 1)")
	genesis.Config.Dbft.BlockPeriod = uint64(w.readDefaultInt(1))

	fmt.Println()
	fmt.Println("how many validators to seal block? (default = 5)")
	vcount := w.readDefaultBigInt(big.NewInt(5))

	fmt.Println()
	fmt.Println("Which accounts to be validators? (mandatory at least one)")


	l := list.New()
	for {
		if address := w.readAddress(); address != nil {
			l.PushBack(*address)
			continue
		}

		if l.Len() > 0 {
			break
		}
	}

	genesis.Alloc[types.VoteContract] = core.GenesisAccount{
		Code: common.Hex2Bytes(voteContractCode),
		Storage: make(map[common.Hash]common.Hash),
		Balance: big.NewInt(1),
	}
	storage := genesis.Alloc[types.VoteContract].Storage

	storage[common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")] = common.BigToHash(vcount) //maxValidators
	listHead := l.Front().Value.(common.Address)
	storage[common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003")] = listHead.Hash() // listHead


	mapHash := func(address common.Address, p uint64) (hash common.Hash) {
		index := make([]byte, 32)
		binary.BigEndian.PutUint64(index, p)

		hasher := sha3.NewKeccak256()
		hasher.Write(abi.U256(address.Big()))
		hasher.Write(abi.U256(new(big.Int).SetUint64(p)))

		hasher.Sum(hash[:0])
		return hash
	}

	candidatesList := func(address common.Address)(prev common.Hash, next common.Hash) {
		hash := mapHash(address, 2)

		hasher := sha3.NewKeccak256()
		hasher.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000"))
		hasher.Write(hash.Bytes())
		hasher.Sum(prev[:0])

		hasher = sha3.NewKeccak256()
		hasher.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"))
		hasher.Write(hash.Bytes())
		hasher.Sum(next[:0])

		return prev, next
	}

	balance, _ := new(big.Int).SetString("1000000000000000000000000000", 10)
	for e := l.Front(); e != nil; e = e.Next() {
		if address, ok := e.Value.(common.Address); ok {
			storage[mapHash(address, 1)] = common.BigToHash(balance) //candidates

			prev, next := candidatesList(address)
			if pe := e.Prev(); pe != nil {
				storage[prev] = pe.Value.(common.Address).Hash() //candidatesList
			}
			if ne := e.Next(); ne != nil {
				storage[next] = ne.Value.(common.Address).Hash() //candidatesList
			}
		}
	}

	fmt.Println()
	fmt.Println("Which accounts should be pre-funded? (advisable at least one)")
	for {
		// Read the address of the account to fund
		if address := w.readAddress(); address != nil {
			genesis.Alloc[*address] = core.GenesisAccount{
				Balance: balance,
			}
			continue
		}
		break
	}

	// All done, store the genesis and flush to disk
	log.Info("Configured new genesis block")

	w.conf.Genesis = genesis
	w.conf.flush()
}