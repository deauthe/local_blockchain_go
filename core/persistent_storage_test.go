package core

import (
	"os"
	"testing"

	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/deauthe/local_blockchain_go/types"
	"github.com/stretchr/testify/assert"
)

func TestPersistentStore(t *testing.T) {
	// Clean up any existing test data
	os.RemoveAll("data")

	// Create new store
	store, err := NewPersistentStore()
	assert.NoError(t, err)
	defer store.Close()

	// Create and store some test blocks
	blocks := make([]*Block, 3)
	for i := 0; i < 3; i++ {
		privKey := crypto.GeneratePrivateKey()
		header := &Header{
			Version:   1,
			Height:    uint32(i),
			Timestamp: 000000,
		}
		block, err := NewBlock(header, nil)
		assert.NoError(t, err)
		assert.NoError(t, block.Sign(privKey))
		blocks[i] = block

		// Store block
		err = store.Put(block)
		assert.NoError(t, err)
	}

	// Test GetAllBlockHashes
	hashes := store.GetAllBlockHashes()
	assert.Equal(t, 3, len(hashes))

	// Test GetBlockByHash for each block
	for _, block := range blocks {
		hash := block.Hash(BlockHasher{})
		retrievedBlock, err := store.GetBlockByHash(hash)
		assert.NoError(t, err)
		assert.Equal(t, block.Height, retrievedBlock.Height)
		assert.Equal(t, block.Hash(BlockHasher{}), retrievedBlock.Hash(BlockHasher{}))
	}

	// Test non-existent hash
	nonExistentHash := types.Hash{}
	_, err = store.GetBlockByHash(nonExistentHash)
	assert.Error(t, err)
}

func TestPersistentStorePersistence(t *testing.T) {
	// Clean up any existing test data
	os.RemoveAll("data")

	// Create first store and add a block
	store1, err := NewPersistentStore()
	assert.NoError(t, err)

	privKey := crypto.GeneratePrivateKey()
	header := &Header{
		Version:   1,
		Height:    0,
		Timestamp: 000000,
	}
	block, err := NewBlock(header, nil)
	assert.NoError(t, err)
	assert.NoError(t, block.Sign(privKey))

	err = store1.Put(block)
	assert.NoError(t, err)
	store1.Close()

	// Create second store and verify block exists
	store2, err := NewPersistentStore()
	assert.NoError(t, err)
	defer store2.Close()

	hash := block.Hash(BlockHasher{})
	retrievedBlock, err := store2.GetBlockByHash(hash)
	assert.NoError(t, err)
	assert.Equal(t, block.Height, retrievedBlock.Height)
	assert.Equal(t, block.Hash(BlockHasher{}), retrievedBlock.Hash(BlockHasher{}))
}

func TestPersistentStoreConcurrent(t *testing.T) {
	// Clean up any existing test data
	os.RemoveAll("data")

	// Create new store
	store, err := NewPersistentStore()
	assert.NoError(t, err)
	defer store.Close()

	// Create a channel to signal when all goroutines are done
	done := make(chan bool)

	// Spawn multiple goroutines to access the store concurrently
	for i := 0; i < 10; i++ {
		go func(i int) {
			// Create and store a block
			privKey := crypto.GeneratePrivateKey()
			header := &Header{
				Version:   1,
				Height:    uint32(i),
				Timestamp: int64(i),
			}
			block, err := NewBlock(header, nil)
			assert.NoError(t, err)
			assert.NoError(t, block.Sign(privKey))

			err = store.Put(block)
			assert.NoError(t, err)

			// Retrieve the block
			hash := block.Hash(BlockHasher{})
			retrievedBlock, err := store.GetBlockByHash(hash)
			assert.NoError(t, err)
			assert.NotNil(t, retrievedBlock)
			assert.Equal(t, block.Hash(BlockHasher{}), retrievedBlock.Hash(BlockHasher{}))

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
