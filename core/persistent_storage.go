package core

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/deauthe/local_blockchain_go/types"
)

type PersistentStore struct {
	mu            sync.RWMutex
	blocksFile    *os.File
	blocksEncoder *gob.Encoder
	blockHashes   map[types.Hash]uint32 // Maps block hash to block height
}

var (
	storeInstance *PersistentStore
	storeOnce     sync.Once
)

// NewPersistentStore creates a new persistent storage instance
func NewPersistentStore() (*PersistentStore, error) {
	var err error
	storeOnce.Do(func() {
		// Create data directory if it doesn't exist
		if err = os.MkdirAll("data", 0755); err != nil {
			err = fmt.Errorf("failed to create data directory: %v", err)
			return
		}

		// Open or create blocks file
		blocksFile, err := os.OpenFile(
			filepath.Join("data", "blocks.gob"),
			os.O_CREATE|os.O_RDWR,
			0644,
		)
		if err != nil {
			err = fmt.Errorf("failed to open blocks file: %v", err)
			return
		}

		storeInstance = &PersistentStore{
			blocksFile:    blocksFile,
			blocksEncoder: gob.NewEncoder(blocksFile),
			blockHashes:   make(map[types.Hash]uint32),
		}

		// Load existing block hashes
		if err = storeInstance.loadBlockHashes(); err != nil {
			blocksFile.Close()
			err = fmt.Errorf("failed to load block hashes: %v", err)
			return
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create persistent storage: %v", err)
	}

	return storeInstance, nil
}

// Put stores a block in the persistent storage
func (s *PersistentStore) Put(b *Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reset file position to end for appending
	if _, err := s.blocksFile.Seek(0, 2); err != nil {
		return fmt.Errorf("failed to seek to end of file: %v", err)
	}

	// Create a new encoder for this write
	encoder := gob.NewEncoder(s.blocksFile)

	// Encode and write block
	if err := encoder.Encode(b); err != nil {
		return fmt.Errorf("failed to encode block: %v", err)
	}

	// Store block hash
	hash := b.Hash(BlockHasher{})
	s.blockHashes[hash] = b.Height

	// Flush to disk
	if err := s.blocksFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync blocks file: %v", err)
	}

	return nil
}

// GetBlockByHash retrieves a block by its hash
func (s *PersistentStore) GetBlockByHash(hash types.Hash) (*Block, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if hash exists
	height, exists := s.blockHashes[hash]
	if !exists {
		return nil, fmt.Errorf("block with hash %s not found", hash)
	}

	// Reset file position to beginning
	if _, err := s.blocksFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to start of file: %v", err)
	}

	// Create a new decoder for this read
	decoder := gob.NewDecoder(s.blocksFile)

	// Read blocks until we find the one with matching height
	var block Block
	for {
		if err := decoder.Decode(&block); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to decode block: %v", err)
		}

		if block.Height == height {
			return &block, nil
		}
	}

	return nil, fmt.Errorf("block with hash %s not found", hash)
}

// GetAllBlockHashes returns all stored block hashes
func (s *PersistentStore) GetAllBlockHashes() map[types.Hash]uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of the map to prevent external modification
	hashes := make(map[types.Hash]uint32)
	for hash, height := range s.blockHashes {
		hashes[hash] = height
	}

	return hashes
}

// Close closes the persistent storage
func (s *PersistentStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.blocksFile.Close(); err != nil {
		return fmt.Errorf("failed to close blocks file: %v", err)
	}

	return nil
}

// loadBlockHashes loads all block hashes from the blocks file
func (s *PersistentStore) loadBlockHashes() error {
	// Reset file position to beginning
	if _, err := s.blocksFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to start of file: %v", err)
	}

	// Create a new decoder for this read
	decoder := gob.NewDecoder(s.blocksFile)

	// Read all blocks and store their hashes
	var block Block
	for {
		if err := decoder.Decode(&block); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to decode block: %v", err)
		}

		hash := block.Hash(BlockHasher{})
		s.blockHashes[hash] = block.Height
	}

	return nil
}
