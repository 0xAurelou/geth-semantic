package vm

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

const (
	// RandomNCSPRNGGasCost is the base cost for the random number generation
	RandomNCSPRNGGasCost = 1024
)

var (
	errInvalidInputLength = errors.New("invalid input length")
	// RandomNCSPRNGContractAddr is the predefined address for the precompile
	RandomNCSPRNGContractAddr = common.HexToAddress("0x6942000000000000000000000000000000000000")
)

// InputParams represents the decoded input parameters
type randomInputParams struct {
	caller common.Address
	n      *big.Int
	nonce  uint64
}

// RandomNCSPRNG implements the PrecompiledContract interface
type randomNCSPRNG struct{}

// RequiredGas returns the gas required to execute the pre-compiled contract
func (r *randomNCSPRNG) RequiredGas(input []byte) uint64 {
	return RandomNCSPRNGGasCost + uint64(len(input)/32)
}

// Run implements the required interface for PrecompiledContract
func (r *randomNCSPRNG) Run(evm *EVM, input []byte) ([]byte, error) {
	// Input format:
	// [0:20]  - caller address
	// [20:52] - n (big.Int, 32 bytes)
	// [52:60] - nonce (uint64, 8 bytes)
	if len(input) < 60 {
		return nil, errInvalidInputLength
	}

	params, err := decodeRandomInput(input)
	if err != nil {
		return nil, err
	}

	nUint256, overflow := uint256.FromBig(params.n)
	if overflow {
		return nil, errors.New("n overflows uint256")
	}

	randomValues, err := generateRandomNCSPRNG(
		RandomNCSPRNGContractAddr,
		params.caller,
		*nUint256,
		params.nonce,
	)
	if err != nil {
		return nil, err
	}

	return encodeRandomOutput(randomValues)
}

// decodeRandomInput extracts parameters from the input bytes
func decodeRandomInput(input []byte) (*randomInputParams, error) {
	caller := common.BytesToAddress(input[:20])
	n := new(big.Int).SetBytes(input[20:52])
	nonce := binary.BigEndian.Uint64(input[52:60])

	return &randomInputParams{
		caller: caller,
		n:      n,
		nonce:  nonce,
	}, nil
}

// encodeRandomOutput encodes the random values into bytes
func encodeRandomOutput(randomValues []*big.Int) ([]byte, error) {
	// Calculate total size: 32 bytes for length + 32 bytes per value
	totalSize := 32 + (32 * len(randomValues))
	output := make([]byte, totalSize)

	// Encode length
	binary.BigEndian.PutUint64(output[24:32], uint64(len(randomValues)))

	// Encode values
	for i, value := range randomValues {
		offset := 32 + (i * 32)
		bytes := value.FillBytes(make([]byte, 32))
		copy(output[offset:offset+32], bytes)
	}

	return output, nil
}

// generateRandomNCSPRNG generates random numbers using HMAC-SHA256
func generateRandomNCSPRNG(precompileAddr common.Address, userAddr common.Address, n uint256.Int, nonce uint64) ([]*big.Int, error) {
	serverSeed := crypto.Keccak256(precompileAddr.Bytes())
	userSeed := crypto.Keccak256(append(userAddr.Bytes(), serverSeed...))

	randomValues := make([]*big.Int, n.Uint64())
	hmac := hmac.New(sha256.New, serverSeed)

	for i := uint64(0); i < n.Uint64(); i++ {
		hmac.Reset()
		hmac.Write(userSeed)

		nonceBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(nonceBytes, nonce)
		hmac.Write(nonceBytes)

		countBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(countBytes, i)
		hmac.Write(countBytes)

		hash := hmac.Sum(nil)
		randomValues[i] = new(big.Int).SetBytes(hash)
	}

	return randomValues, nil
}
