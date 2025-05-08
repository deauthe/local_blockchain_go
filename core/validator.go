package core

import (
	"errors"
	"fmt"
)

var ErrBlockKnown = errors.New("block already known")

type Validator interface {
	ValidateBlock(*Block) error
}

type BlockValidator struct {
	bc *Blockchain
}

func NewBlockValidator(bc *Blockchain) *BlockValidator {
	return &BlockValidator{
		bc: bc,
	}
}

func (v *BlockValidator) ValidateBlock(b *Block) error {
	currentHeight := v.bc.Height()

	if v.bc.HasBlock(b.Height) {
		return ErrBlockKnown
	}

	// Allow blocks that are at most 1 height ahead
	if b.Height > currentHeight+1 {
		return fmt.Errorf("block height too high: got %d, current height is %d", b.Height, currentHeight)
	}

	// If this is not the genesis block, validate the previous block hash
	if b.Height > 0 {
		// Get the previous header for validation
		prevHeader, err := v.bc.GetHeader(b.Height - 1)
		if err != nil {
			return fmt.Errorf("failed to get previous header: %v", err)
		}

		hash := BlockHasher{}.Hash(prevHeader)
		if hash != b.PrevBlockHash {
			return fmt.Errorf("invalid previous block hash: expected %s, got %s",
				hash, b.PrevBlockHash)
		}
	}

	// Verify block signature and transactions
	if err := b.Verify(); err != nil {
		return fmt.Errorf("block verification failed: %v", err)
	}

	// Verify data hash
	dataHash, err := CalculateDataHash(b.Transactions)
	if err != nil {
		return fmt.Errorf("failed to calculate data hash: %v", err)
	}
	if dataHash != b.DataHash {
		return fmt.Errorf("invalid data hash: expected %s, got %s",
			dataHash, b.DataHash)
	}

	return nil
}
