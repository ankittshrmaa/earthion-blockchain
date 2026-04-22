package core

// Reward constants (BTC-style halving)
const (
	InitialReward    = 50      // Initial block reward
	HalvingInterval  = 210000  // Blocks per halving (BTC: 210,000)
	MinReward       = 1        // Minimum reward (dust limit)
)

// GetBlockReward returns the reward for a given block height
// Implements halving every HalvingInterval blocks
func GetBlockReward(blockHeight int) int {
	// Calculate how many halvings have occurred
	halvings := blockHeight / HalvingInterval

	// Calculate reward: InitialReward / 2^halvings
	reward := InitialReward
	for i := 0; i < halvings; i++ {
		reward /= 2
		if reward < MinReward {
			return MinReward
		}
	}

	// Ensure we don't go below minimum
	if reward < MinReward {
		return MinReward
	}

	return reward
}

// GetTotalMined returns the total coins that should exist at given height
// Only accounts for block rewards (not fees)
func GetTotalMined(blockHeight int) int {
	total := 0

	for h := 0; h <= blockHeight; h++ {
		total += GetBlockReward(h)
	}

	return total
}

// GetMaxSupply returns the theoretical maximum supply
// Sum of all rewards from genesis to infinity (geometric series)
// InitialReward * 2 / 1 = InitialReward * 2 for infinite halvings
// But practically: InitialReward * 2 (since halving never reaches true 0)
func GetMaxSupply() int {
	// With halving every 210k blocks, max supply is:
	// 50 * 210000 + 25 * 210000 + 12.5 * 210000 + ...
	// This converges to 50 * 2 * 210000 = 21,000,000
	return InitialReward * 2 * HalvingInterval
}