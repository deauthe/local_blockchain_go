package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/deauthe/local_blockchain_go/types"
	"github.com/go-kit/log"
)

type Blockchain struct {
	logger log.Logger
	store  Storage
	// TODO: double check this!
	lock       sync.RWMutex
	Headers    []*Header
	blocks     []*Block
	txStore    map[types.Hash]*Transaction
	blockStore map[types.Hash]*Block

	AccountState *AccountState

	stateLock       sync.RWMutex
	collectionState map[types.Hash]*CollectionTx
	mintState       map[types.Hash]*MintTx
	studentState    map[string]*Student
	Validator       Validator
	// TODO: make this an interface.
	contractState *State
}

func NewBlockchain(logger log.Logger, genesis *Block) (*Blockchain, error) {
	// We should create all states inside the scope of the newblockchain.

	// TODO: read this from disk later on
	accountState := NewAccountState()

	coinbase := crypto.PublicKey{}
	accountState.CreateAccount(coinbase.Address())

	// Create persistent storage
	store, err := NewPersistentStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create persistent storage: %v", err)
	}

	bc := &Blockchain{
		contractState:   NewState(),
		Headers:         []*Header{},
		store:           store,
		logger:          logger,
		AccountState:    accountState,
		collectionState: make(map[types.Hash]*CollectionTx),
		mintState:       make(map[types.Hash]*MintTx),
		studentState:    make(map[string]*Student),
		blockStore:      make(map[types.Hash]*Block),
		txStore:         make(map[types.Hash]*Transaction),
	}
	bc.Validator = NewBlockValidator(bc)
	err = bc.addBlockWithoutValidation(genesis)

	return bc, err
}

func (bc *Blockchain) SetValidator(v Validator) {
	bc.Validator = v
}

func (bc *Blockchain) AddBlock(b *Block) error {
	if err := bc.Validator.ValidateBlock(b); err != nil {
		return err
	}

	return bc.addBlockWithoutValidation(b)
}

func (bc *Blockchain) handleNativeTransfer(tx *Transaction) error {
	bc.logger.Log(
		"msg", "handle native token transfer",
		"from", tx.From,
		"to", tx.To,
		"value", tx.Value)

	return bc.AccountState.Transfer(tx.From.Address(), tx.To.Address(), tx.Value)
}

func (bc *Blockchain) handleNativeNFT(tx *Transaction) error {
	hash := tx.Hash(TxHasher{})

	switch t := tx.TxInner.(type) {
	case CollectionTx:
		bc.collectionState[hash] = &t
		bc.logger.Log("msg", "created new NFT collection", "hash", hash)
	case MintTx:
		_, ok := bc.collectionState[t.Collection]
		if !ok {
			return fmt.Errorf("collection (%s) does not exist on the blockchain", t.Collection)
		}
		bc.mintState[hash] = &t

		bc.logger.Log("msg", "created new NFT mint", "NFT", t.NFT, "collection", t.Collection)
	default:
		return fmt.Errorf("unsupported tx type %v", t)
	}

	return nil
}

func (bc *Blockchain) GetBlockByHash(hash types.Hash) (*Block, error) {
	// First check the in-memory cache
	bc.lock.RLock()
	block, ok := bc.blockStore[hash]
	bc.lock.RUnlock()
	if ok {
		return block, nil
	}

	// If not in cache, try persistent storage
	if persistentStore, ok := bc.store.(*PersistentStore); ok {
		return persistentStore.GetBlockByHash(hash)
	}

	return nil, fmt.Errorf("block with hash (%s) not found", hash)
}

func (bc *Blockchain) GetBlock(height uint32) (*Block, error) {
	if height > bc.Height() {
		return nil, fmt.Errorf("given height (%d) too high", height)
	}

	bc.lock.Lock()
	defer bc.lock.Unlock()

	return bc.blocks[height], nil
}

func (bc *Blockchain) GetHeader(height uint32) (*Header, error) {
	if height > bc.Height() {
		return nil, fmt.Errorf("given height (%d) too high", height)
	}

	bc.lock.Lock()
	defer bc.lock.Unlock()

	return bc.Headers[height], nil
}

func (bc *Blockchain) GetTxByHash(hash types.Hash) (*Transaction, error) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	tx, ok := bc.txStore[hash]
	if !ok {
		return nil, fmt.Errorf("could not find tx with hash (%s)", hash)
	}

	return tx, nil
}

func (bc *Blockchain) HasBlock(height uint32) bool {
	return height <= bc.Height()
}

// [0, 1, 2 ,3] => 4 len
// [0, 1, 2 ,3] => 3 height
func (bc *Blockchain) Height() uint32 {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return uint32(len(bc.Headers) - 1)
}

func (bc *Blockchain) handleStudentTx(tx *Transaction) error {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		bc.logger.Log("msg", fmt.Sprintf("Handled student transaction for ID: %s (took %v)", tx.TxInner.(StudentTx).StudentID, elapsed))
	}()

	studentTx, ok := tx.TxInner.(StudentTx)
	if !ok {
		return fmt.Errorf("invalid student transaction type")
	}

	bc.stateLock.Lock()
	defer bc.stateLock.Unlock()

	switch studentTx.Type {
	case StudentTxTypeCreate:
		if _, exists := bc.studentState[studentTx.StudentID]; exists {
			return fmt.Errorf("student with ID %s already exists", studentTx.StudentID)
		}
		bc.studentState[studentTx.StudentID] = studentTx.Student
		elapsed := time.Since(startTime)
		bc.logger.Log("msg", fmt.Sprintf("Created new student with ID: %s (took %v)", studentTx.StudentID, elapsed))

	case StudentTxTypeUpdate:
		if _, exists := bc.studentState[studentTx.StudentID]; !exists {
			return fmt.Errorf("student with ID %s does not exist", studentTx.StudentID)
		}
		bc.studentState[studentTx.StudentID] = studentTx.Student
		elapsed := time.Since(startTime)
		bc.logger.Log("msg", fmt.Sprintf("Updated student with ID: %s (took %v)", studentTx.StudentID, elapsed))

	case StudentTxTypeDelete:
		if _, exists := bc.studentState[studentTx.StudentID]; !exists {
			return fmt.Errorf("student with ID %s does not exist", studentTx.StudentID)
		}
		delete(bc.studentState, studentTx.StudentID)
		elapsed := time.Since(startTime)
		bc.logger.Log("msg", fmt.Sprintf("Deleted student with ID: %s (took %v)", studentTx.StudentID, elapsed))
	}

	return nil
}

func (bc *Blockchain) handleTransaction(tx *Transaction) error {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		bc.logger.Log("msg", fmt.Sprintf("Handled transaction with hash: %s (took %v)", tx.Hash(TxHasher{}), elapsed))
	}()

	// If we have data inside execute that data on the VM.
	if len(tx.Data) > 0 {
		bc.logger.Log("msg", fmt.Sprintf("Executing code, len: %d, hash: %s", len(tx.Data), tx.Hash(TxHasher{})))

		vm := NewVM(tx.Data, bc.contractState)
		if err := vm.Run(); err != nil {
			return err
		}
	}

	// Handle student transactions
	if _, ok := tx.TxInner.(StudentTx); ok {
		return bc.handleStudentTx(tx)
	}

	// If the txInner of the transaction is not nil we need to handle
	// the native NFT implemtation.
	if tx.TxInner != nil {
		if err := bc.handleNativeNFT(tx); err != nil {
			return err
		}
	}

	// Handle the native transaction here
	if tx.Value > 0 {
		if err := bc.handleNativeTransfer(tx); err != nil {
			return err
		}
	}

	return nil
}

func (bc *Blockchain) addBlockWithoutValidation(b *Block) error {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		bc.logger.Log("msg", fmt.Sprintf("Added block with hash: %s, height: %d, transactions: %d (took %v)",
			b.Hash(BlockHasher{}), b.Height, len(b.Transactions), elapsed))
	}()

	bc.stateLock.Lock()
	for i := 0; i < len(b.Transactions); i++ {
		if err := bc.handleTransaction(b.Transactions[i]); err != nil {
			bc.logger.Log("msg", fmt.Sprintf("Error handling transaction: %v", err))

			b.Transactions[i] = b.Transactions[len(b.Transactions)-1]
			b.Transactions = b.Transactions[:len(b.Transactions)-1]

			continue
		}
	}
	bc.stateLock.Unlock()

	bc.lock.Lock()
	bc.Headers = append(bc.Headers, b.Header)
	bc.blocks = append(bc.blocks, b)
	bc.blockStore[b.Hash(BlockHasher{})] = b

	for _, tx := range b.Transactions {
		bc.txStore[tx.Hash(TxHasher{})] = tx
	}
	bc.lock.Unlock()

	return bc.store.Put(b)
}

// GetAllBlockHashes returns a map of all block hashes to their heights
func (bc *Blockchain) GetAllBlockHashes() map[types.Hash]uint32 {
	if persistentStore, ok := bc.store.(*PersistentStore); ok {
		return persistentStore.GetAllBlockHashes()
	}
	return nil
}
