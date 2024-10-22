package vm

import (
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// randomContractAddr defines the precompile contract address for `random` precompile
var randomPRNGContractAddr = common.HexToAddress("0x0000000000000000000000000000000000069420")

type randomPRNG struct{}

func (p *randomPRNG) RequiredGas(input []byte) uint64 {
	return uint64(1024)
}

var randomPRNGABI = `[
  {
    "type": "function",
    "name": "randomPRNG",
    "inputs": [],
    "outputs": [
      {
        "name": "randomValue",
        "type": "uint256",
        "internalType": "uint256"
      }
    ],
    "stateMutability": "view"
  }
]`

// parseABI parses the abijson string and returns the parsed abi object.
func parseABI(abiJSON string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic(err)
	}

	return parsed
}

func packRandomPRNGOutput(result *big.Int) ([]byte, error) {
	parsedABI := parseABI(randomPRNGABI)
	return parsedABI.Methods["randomPRNG"].Outputs.Pack(result)
}

// getRandomNumber generates a pseudo-random big.Int within the range of int64 using math/rand
func getRandomNumber(blockNumber uint64) *big.Int {
	// Seed the random generator with blockNumber for deterministic randomness
	source := rand.NewSource(int64(blockNumber))
	rng := rand.New(source)

	// Generate a random number in the range of [0, math.MaxInt64]
	randomValue := rng.Int63()

	return big.NewInt(randomValue)
}

func (p *randomPRNG) Run(input []byte) ([]byte, error) {
	if len(input) < 4 {
		return nil, fmt.Errorf("Function selector is missing")
	}

	// Get the block number (you would need to pass this from the EVM context)
	blockNumber := uint64(time.Now().Unix()) // Placeholder for block number, replace in a real environment

	// Generate a random number based on the block number
	resultBigInt := getRandomNumber(blockNumber)

	// Pack resultBigInt into output
	output, err := packRandomPRNGOutput(resultBigInt)
	if err != nil {
		return nil, err
	}

	return output, nil
}
