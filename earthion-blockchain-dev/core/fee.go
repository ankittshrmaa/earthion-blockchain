package core

import "encoding/hex"

// Fee constants
const (
	MinFeeRate      = 1   // Minimum fee rate (sat/byte equivalent)
	DefaultFeeRate = 10  // Default fee rate
	MaxFeeRate     = 1000 // Maximum fee rate (prevent abuse)
)

// FeeCalculator handles transaction fee calculations
type FeeCalculator struct {
	FeeRate int // fee per byte
}

// NewFeeCalculator creates a new fee calculator with default rate
func NewFeeCalculator() *FeeCalculator {
	return &FeeCalculator{FeeRate: DefaultFeeRate}
}

// SetFeeRate sets the fee rate (with bounds checking)
func (fc *FeeCalculator) SetFeeRate(rate int) {
	if rate < MinFeeRate {
		fc.FeeRate = MinFeeRate
	} else if rate > MaxFeeRate {
		fc.FeeRate = MaxFeeRate
	} else {
		fc.FeeRate = rate
	}
}

// CalculateFee calculates the fee for a transaction based on its size
func (fc *FeeCalculator) CalculateFee(tx *Transaction) int {
	size := len(tx.Serialize())
	return size * fc.FeeRate
}

// CalculateFeeForInputsOutputs calculates fee based on input/output count
// This is an approximation that doesn't require full serialization
func (fc *FeeCalculator) CalculateFeeForInputsOutputs(inputs, outputs int) int {
	// Rough size estimation:
	// - Input: ~41 bytes (TXID 32 + OutIndex 4 + Sig ~65 + PubKey ~33 = ~140, but compressed)
	// - Output: ~8 bytes (Value 8 + PubKeyHash 20 = ~28)
	// Base transaction overhead: ~10 bytes
	estimatedSize := 10 + (inputs * 140) + (outputs * 28)
	return estimatedSize * fc.FeeRate
}

// GetFeeRate returns the current fee rate
func (fc *FeeCalculator) GetFeeRate() int {
	return fc.FeeRate
}

// CalculateTotalInputs calculates total input value from UTXO references
func CalculateTotalInputs(tx *Transaction) int {
	total := 0
	for _, in := range tx.Inputs {
		if in.OutIndex == -1 {
			continue // coinbase has no real input value
		}
		// Note: This is just a placeholder - actual value lookup requires UTXO index
		// The actual fee deduction happens in NewTransaction when change is calculated
	}
	return total
}

// CalculateTotalOutputs calculates total output value
func CalculateTotalOutputs(tx *Transaction) int {
	total := 0
	for _, out := range tx.Outputs {
		total += out.Value
	}
	return total
}

// GetTxFee calculates the fee that would be paid for a transaction
// Returns input total - output total (fees go to miner)
func GetTxFee(tx *Transaction, bc *Blockchain) int {
	inputTotal := 0

	// Look up input values from UTXO set
	for _, in := range tx.Inputs {
		if in.OutIndex == -1 {
			continue // coinbase
		}

		txID := hex.EncodeToString(in.Txid)
		key := txID + ":" + itoa(in.OutIndex)
		if utxo, exists := bc.UTXOIndex()[key]; exists {
			inputTotal += utxo.Value
		}
	}

	outputTotal := CalculateTotalOutputs(tx)
	return inputTotal - outputTotal
}

// IsFeeSufficient checks if the fee is adequate for network conditions
func (fc *FeeCalculator) IsFeeSufficient(tx *Transaction) bool {
	calculatedFee := fc.CalculateFee(tx)
	txFee := CalculateTotalInputs(tx) - CalculateTotalOutputs(tx)
	return txFee >= calculatedFee
}