// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package ethereum

import (
	"context"
	"math/big"
	"testing"

	ethutils "github.com/ChainSafe/ChainBridge/shared/ethereum"
	ethtest "github.com/ChainSafe/ChainBridge/shared/ethereum/testing"
	"github.com/ChainSafe/chainbridge-utils/keystore"
	"github.com/ChainSafe/log15"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

var TestEndpoint = "ws://localhost:8545"
var AliceKp = keystore.TestKeyRing.EthereumKeys[keystore.AliceKey]
var GasLimit = big.NewInt(ethutils.DefaultGasLimit)
var MaxGasPrice = big.NewInt(ethutils.DefaultMaxGasPrice)

var GasMultipler = big.NewFloat(ethutils.DefaultGasMultiplier)

func TestConnect(t *testing.T) {
	conn := NewConnection(TestEndpoint, nil, false, AliceKp, log15.Root(), GasLimit, MaxGasPrice, GasMultipler, "", "")
	err := conn.Connect()
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

// TestContractCode is used to make sure the contracts are deployed correctly.
// This is probably the least intrusive way to check if the contracts exists
func TestContractCode(t *testing.T) {
	client := ethtest.NewClient(t, TestEndpoint, AliceKp)
	contracts, err := ethutils.DeployContracts(client, 0, big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}

	conn := NewConnection(TestEndpoint, nil, false, AliceKp, log15.Root(), GasLimit, MaxGasPrice, GasMultipler, "", "")
	err = conn.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// The following section checks if the byteCode exists on the chain at the specificed Addresses
	err = conn.EnsureHasBytecode(contracts.BridgeAddress)
	if err != nil {
		t.Fatal(err)
	}

	err = conn.EnsureHasBytecode(ethcmn.HexToAddress("0x0"))
	if err == nil {
		t.Fatal("should detect no bytecode")
	}

}

func TestConnection_SafeEstimateGas(t *testing.T) {
	// MaxGasPrice is the constant price on the dev network, so we increase it here by 1 to ensure it adjusts
	conn := NewConnection(TestEndpoint, nil, false, AliceKp, log15.Root(), GasLimit, MaxGasPrice.Add(MaxGasPrice, big.NewInt(1)), GasMultipler, "", "")
	err := conn.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	price, err := conn.SafeEstimateGas(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if price.Cmp(MaxGasPrice) == 0 {
		t.Fatalf("Gas price should be less than max. Suggested: %s Max: %s", price.String(), MaxGasPrice.String())
	}
}

func TestConnection_SafeEstimateGasMax(t *testing.T) {
	maxPrice := big.NewInt(1)
	conn := NewConnection(TestEndpoint, nil, false, AliceKp, log15.Root(), GasLimit, maxPrice, GasMultipler, "", "")
	err := conn.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	price, err := conn.SafeEstimateGas(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if price.Cmp(maxPrice) != 0 {
		t.Fatalf("Gas price should equal max. Suggested: %s Max: %s", price.String(), maxPrice.String())
	}
}

func TestConnection_EstimateGasLondon(t *testing.T) {
	// Set TestEndpoint to Goerli endpoint when testing as the current Github CI doesn't use the London version of geth
	// Goerli commonly has a base fee of 7 gwei with maxPriorityFeePerGas of 4.999999993 gwei
	maxGasPrice := big.NewInt(100000000000)
	conn := NewConnection(TestEndpoint, false, AliceKp, log15.Root(), GasLimit, maxGasPrice, GasMultipler, "", "")
	err := conn.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	head, err := conn.conn.HeaderByNumber(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// This is here as the current dev network is an old version of geth and will keep the test failing on the CI
	if head.BaseFee != nil {
		_, suggestedGasFeeCap, err := conn.EstimateGasLondon(context.Background(), head.BaseFee)
		if err != nil {
			t.Fatal(err)
		}

		if suggestedGasFeeCap.Cmp(maxGasPrice) >= 0 {
			t.Fatalf("Gas fee cap should be less than max gas price. Suggested: %s Max: %s", suggestedGasFeeCap.String(), maxGasPrice.String())
		}
	}
}

func TestConnection_EstimateGasLondonMax(t *testing.T) {
	// Set TestEndpoint to Goerli endpoint when testing as the current Github CI doesn't use the London version of geth
	// Goerli commonly has a base fee of 7 gwei with maxPriorityFeePerGas of 4.999999993 gwei
	maxGasPrice := big.NewInt(100)
	conn := NewConnection(TestEndpoint, false, AliceKp, log15.Root(), GasLimit, maxGasPrice, GasMultipler, "", "")
	err := conn.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	head, err := conn.conn.HeaderByNumber(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// This is here as the current dev network is an old version of geth and will keep the test failing on the CI
	if head.BaseFee != nil {
		suggestedGasTip, suggestedGasFeeCap, err := conn.EstimateGasLondon(context.Background(), head.BaseFee)
		if err != nil {
			t.Fatal(err)
		}

		maxPriorityFeePerGas := new(big.Int).Sub(maxGasPrice, head.BaseFee)
		if suggestedGasTip.Cmp(maxPriorityFeePerGas) != 0 {
			t.Fatalf("Gas tip cap should equal max - baseFee. Suggested: %s Max Tip: %s", suggestedGasTip.String(), maxPriorityFeePerGas.String())
		}

		if suggestedGasFeeCap.Cmp(maxGasPrice) != 0 {
			t.Fatalf("Gas fee cap should equal max gas price. Suggested: %s Max: %s", suggestedGasFeeCap.String(), maxGasPrice.String())
		}

	}
}

func TestConnection_EstimateGasLondonMin(t *testing.T) {
	// Set TestEndpoint to Goerli endpoint when testing as the current Github CI doesn't use the London version of geth
	// Goerli commonly has a base fee of 7 gwei with maxPriorityFeePerGas of 4.999999993 gwei
	maxGasPrice := big.NewInt(1)
	conn := NewConnection(TestEndpoint, false, AliceKp, log15.Root(), GasLimit, maxGasPrice, GasMultipler, "", "")
	err := conn.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	head, err := conn.conn.HeaderByNumber(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	// This is here as the current dev network is an old version of geth and will keep the test failing on the CI
	if head.BaseFee != nil {
		suggestedGasTip, suggestedGasFeeCap, err := conn.EstimateGasLondon(context.Background(), head.BaseFee)
		if err != nil {
			t.Fatal(err)
		}

		maxPriorityFeePerGas := big.NewInt(1)
		maxFeePerGas := new(big.Int).Add(head.BaseFee, maxPriorityFeePerGas)

		if suggestedGasTip.Cmp(maxPriorityFeePerGas) != 0 {
			t.Fatalf("Gas tip cap should be equal to 1. Suggested: %s Max Tip: %s", suggestedGasTip.String(), maxPriorityFeePerGas)
		}

		if suggestedGasFeeCap.Cmp(maxFeePerGas) != 0 {
			t.Fatalf("Gas fee cap should be 1 greater than the base fee. Suggested: %s Max: %s", suggestedGasFeeCap.String(), maxFeePerGas.String())
		}
	}
}
