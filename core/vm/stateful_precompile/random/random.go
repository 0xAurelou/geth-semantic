// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package random

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm/stateful_precompile/contract"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

const (
	RandomNCSPRNGGasCost = 1024
)

var (
	errInvalidInputLength = errors.New("invalid input length")
	randomNCSPRNGABI      = `[
	  {
		"type": "function",
		"name": "randomNCSPRNG",
		"inputs": [
		  {
			"name": "n",
			"type": "uint256",
			"internalType": "uint256"
		  }
		],
		"outputs": [
		  {
			"name": "randomValues",
			"type": "uint256[]",
			"internalType": "uint256[]"
		  }
		],
		"stateMutability": "view"
	  }
	]`
)

var randomNCSPRNGContractAddr = common.HexToAddress("0x6942000000000000000000000000000000000000")

func PackRandomNCSPRNGInput(n *big.Int) ([]byte, error) {
	abi := contract.ParseABI(randomNCSPRNGABI)
	return abi.Pack("randomNCSPRNG", n)
}

func UnpackRandomNCSPRNGInput(input []byte) (*big.Int, error) {
	if len(input) != 32 {
		return nil, errInvalidInputLength
	}
	return new(big.Int).SetBytes(input), nil
}

func PackRandomNCSPRNGOutput(randomValues []*big.Int) ([]byte, error) {
	abi := contract.ParseABI(randomNCSPRNGABI)
	return abi.Methods["randomNCSPRNG"].Outputs.Pack(randomValues)
}

func generateRandomNCSPRNG(precompileAddr common.Address, userAddr common.Address, n uint256.Int, state contract.StateDB) ([]*big.Int, error) {
	serverSeed := crypto.Keccak256(precompileAddr.Bytes())
	userSeed := crypto.Keccak256(append(userAddr.Bytes(), serverSeed...))
	nonce := state.GetNonce(userAddr)

	randomValues := make([]*big.Int, n.Uint64())
	hmac := hmac.New(sha256.New, serverSeed)
	for i := uint64(0); i < n.Uint64(); i++ {
		hmac.Reset()
		hmac.Write(userSeed)
		hmac.Write(common.BigToHash(new(big.Int).SetUint64(nonce)).Bytes())
		hmac.Write(common.BigToHash(new(big.Int).SetUint64(i)).Bytes())
		hash := hmac.Sum(nil)
		randomValues[i] = new(big.Int).SetBytes(hash)
	}

	return randomValues, nil
}

func RandomNCSPRNGFunc(accessibleState contract.AccessibleState, caller common.Address, addr common.Address, input []byte, suppliedGas uint64, readOnly bool) (ret []byte, remainingGas uint64, err error) {
	if remainingGas, err = contract.DeductGas(suppliedGas, RandomNCSPRNGGasCost); err != nil {
		return nil, 0, err
	}

	n, err := UnpackRandomNCSPRNGInput(input)
	if err != nil {
		return nil, remainingGas, err
	}

	nUint256, overflow := uint256.FromBig(n)
	if overflow {
		return nil, remainingGas, errors.New("n overflows uint256")
	}

	randomValues, err := generateRandomNCSPRNG(addr, caller, *nUint256, accessibleState.GetStateDB())
	if err != nil {
		return nil, remainingGas, err
	}

	ret, err = PackRandomNCSPRNGOutput(randomValues)
	if err != nil {
		return nil, remainingGas, err
	}

	return ret, remainingGas, nil
}

// CreateRandomNCSPRNGPrecompile returns a StatefulPrecompiledContract with randomNCSPRNG function
func CreateRandomNCSPRNGPrecompile() contract.StatefulPrecompiledContract {
	abi := contract.ParseABI(randomNCSPRNGABI)

	randomNCSPRNGFunction := contract.NewStatefulPrecompileFunction(abi.Methods["randomNCSPRNG"].ID, RandomNCSPRNGFunc)
	contract, err := contract.NewStatefulPrecompileContract(nil, []*contract.StatefulPrecompileFunction{randomNCSPRNGFunction})
	if err != nil {
		panic(err)
	}
	return contract
}
