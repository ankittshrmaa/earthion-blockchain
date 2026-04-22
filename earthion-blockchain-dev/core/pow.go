package core

import (
	"bytes"
	"log"
	"math"
	"math/big"

	"earthion/crypto"
)

const (
	InitialDifficulty    = 18
	TargetBlockTime      = 10 // seconds - desired block time
	AdjustmentInterval   = 10 // adjust difficulty every N blocks
	MinDifficulty        = 1  // Minimum difficulty (easiest allowed)
	MaxDifficulty        = 26 // Maximum difficulty - tighter bound to prevent gaming (was 32)
	MaxDifficultyChange  = 6  // Increased from 4 for faster adaptation
)

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

// CurrentDifficulty returns the current difficulty for mining a new block
// Uses a smoothed adjustment algorithm to prevent gaming
func CurrentDifficulty(chain []*Block) uint32 {
	if len(chain) < AdjustmentInterval+1 {
		return InitialDifficulty
	}

	// Calculate time taken for last N blocks
	oldBlock := chain[len(chain)-AdjustmentInterval]
	newBlock := chain[len(chain)-1]
	timeDiff := newBlock.TimestampInt() - oldBlock.TimestampInt()

	// Handle case where timestamps are too close or identical
	if timeDiff <= 0 {
		timeDiff = 1
	}

	// Expected time = N blocks * target block time
	expectedTime := int64(AdjustmentInterval * TargetBlockTime)

	// If blocks are coming too fast, increase difficulty
	// If blocks are too slow, decrease difficulty
	ratio := float64(expectedTime) / float64(timeDiff)

	currentDiff := chain[len(chain)-1].Difficulty

	var newDiff uint32

	// Enhanced adjustment algorithm with smoother transitions
	// Use logarithm for smoother scaling of extreme ratios
	if ratio <= 0.1 {
		// Extremely fast blocks (>10x expected) - maximum increase
		newDiff = currentDiff + MaxDifficultyChange*2
		log.Printf("[DIFFICULTY] Extreme speed detected (ratio: %.2f), increasing by max", ratio)
	} else if ratio <= 0.25 {
		// Very fast blocks (4-10x) - significant increase
		newDiff = currentDiff + MaxDifficultyChange
	} else if ratio <= 0.5 {
		// Fast blocks (2-4x) - moderate increase
		newDiff = currentDiff + MaxDifficultyChange/2
	} else if ratio <= 0.75 {
		// Slightly fast (1.33-2x) - small increase
		newDiff = currentDiff + 1
	} else if ratio >= 10.0 {
		// Extremely slow (<0.1x expected) - maximum decrease
		if currentDiff >= MaxDifficultyChange*2+1 {
			newDiff = currentDiff - MaxDifficultyChange*2
		} else if currentDiff >= MaxDifficultyChange+1 {
			newDiff = currentDiff - MaxDifficultyChange
		} else if currentDiff >= 2 {
			newDiff = currentDiff - 1
		} else {
			newDiff = currentDiff
		}
		log.Printf("[DIFFICULTY] Extreme slowness detected (ratio: %.2f), decreasing by max", ratio)
	} else if ratio >= 4.0 {
		// Very slow (0.25x) - significant decrease
		if currentDiff >= MaxDifficultyChange+1 {
			newDiff = currentDiff - MaxDifficultyChange
		} else if currentDiff >= 2 {
			newDiff = currentDiff - 1
		} else {
			newDiff = currentDiff
		}
	} else if ratio >= 2.0 {
		// Slow (0.5x) - moderate decrease
		if currentDiff >= MaxDifficultyChange/2+1 {
			newDiff = currentDiff - MaxDifficultyChange/2
		} else if currentDiff >= 2 {
			newDiff = currentDiff - 1
		} else {
			newDiff = currentDiff
		}
	} else if ratio >= 1.25 {
		// Slightly slow (0.8x) - small decrease
		if currentDiff >= 2 {
			newDiff = currentDiff - 1
		} else {
			newDiff = currentDiff
		}
	} else if ratio >= 0.9 && ratio <= 1.1 {
		// Near target - no change (within 10% of target)
		newDiff = currentDiff
	} else {
		// Default: keep current
		newDiff = currentDiff
	}

	// Enforce difficulty bounds
	if newDiff < MinDifficulty {
		newDiff = MinDifficulty
	}
	if newDiff > MaxDifficulty {
		newDiff = MaxDifficulty
	}

	// Log significant changes
	if newDiff != currentDiff {
		log.Printf("[DIFFICULTY] Adjusted: %d -> %d (ratio: %.2f, time: %ds/%ds)",
			currentDiff, newDiff, ratio, timeDiff, expectedTime)
	}

	return newDiff
}

func NewProofOfWork(b *Block) *ProofOfWork {
	difficulty := b.Difficulty
	if difficulty == 0 {
		difficulty = InitialDifficulty
	}
	// Enforce bounds in case loaded from storage with invalid value
	if difficulty < MinDifficulty {
		difficulty = MinDifficulty
	}
	if difficulty > MaxDifficulty {
		difficulty = MaxDifficulty
	}

	target := big.NewInt(1)
	target.Lsh(target, 256-uint(difficulty))
	return &ProofOfWork{b, target}
}

// prepareData creates the data to be hashed for PoW
// Includes all critical block fields for mining security
func (pow *ProofOfWork) prepareData(nonce int) []byte {
	txData := []byte{}

	for _, tx := range pow.Block.Transactions {
		txData = append(txData, tx.Serialize()...)
	}

	return bytes.Join(
		[][]byte{
			IntToHex(int64(pow.Block.Index)),
			IntToHex(pow.Block.TimestampInt()),
			pow.Block.PrevHash,
			pow.Block.MerkleRoot,
			IntToHex(int64(pow.Block.Difficulty)),
			txData,
			IntToHex(int64(nonce)),
		},
		[]byte{},
	)
}

func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash []byte
	nonce := 0

	for nonce < math.MaxInt64 {
		data := pow.prepareData(nonce)
		hash = crypto.DoubleHash(data)

		hashInt.SetBytes(hash)

		if hashInt.Cmp(pow.Target) == -1 {
			break
		}
		nonce++
	}

	return nonce, hash
}

// Validate verifies the proof of work
func (pow *ProofOfWork) Validate() bool {
	data := pow.prepareData(pow.Block.Nonce)
	hash := crypto.DoubleHash(data)

	// Verify the calculated hash matches the stored block hash
	if !bytes.Equal(hash, pow.Block.Hash) {
		return false
	}

	// Also verify hash meets difficulty target
	var hashInt big.Int
	hashInt.SetBytes(hash)

	return hashInt.Cmp(pow.Target) == -1
}