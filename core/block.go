package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/deauthe/local_blockchain_go/types"
)

type Header struct {
	Version       uint32
	DataHash      types.Hash
	PrevBlockHash types.Hash
	Height        uint32
	Timestamp     int64
}

func (h *Header) Bytes() []byte {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	enc.Encode(h)

	return buf.Bytes()
}

type Block struct {
	*Header

	Transactions []*Transaction
	Validator    crypto.PublicKey
	Signature    *crypto.Signature

	// Cached version of the header hash
	hash types.Hash
}

func init() {
	// Register types for GOB encoding/decoding
	gob.Register(&Block{})
	gob.Register(&Header{})
	gob.Register(&Transaction{})
	gob.Register(types.Hash{})
	gob.Register(&crypto.PublicKey{})
	gob.Register(&crypto.Signature{})
}

func NewBlock(h *Header, txx []*Transaction) (*Block, error) {
	return &Block{
		Header:       h,
		Transactions: txx,
	}, nil
}

func NewBlockFromPrevHeader(prevHeader *Header, txx []*Transaction) (*Block, error) {
	dataHash, err := CalculateDataHash(txx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate data hash: %v", err)
	}

	header := &Header{
		Version:       1,
		Height:        prevHeader.Height + 1,
		DataHash:      dataHash,
		PrevBlockHash: BlockHasher{}.Hash(prevHeader),
		Timestamp:     time.Now().UnixNano(),
	}

	block, err := NewBlock(header, txx)
	if err != nil {
		return nil, fmt.Errorf("failed to create block: %v", err)
	}

	// Verify the block's data hash matches
	verifyHash, err := CalculateDataHash(block.Transactions)
	if err != nil {
		return nil, fmt.Errorf("failed to verify data hash: %v", err)
	}
	if verifyHash != block.DataHash {
		return nil, fmt.Errorf("data hash mismatch: expected %s, got %s",
			verifyHash, block.DataHash)
	}

	return block, nil
}

func (b *Block) AddTransaction(tx *Transaction) {
	b.Transactions = append(b.Transactions, tx)
	hash, _ := CalculateDataHash(b.Transactions)
	b.DataHash = hash
}

func (b *Block) Sign(privKey crypto.PrivateKey) error {
	sig, err := privKey.Sign(b.Header.Bytes())
	if err != nil {
		return err
	}

	b.Validator = privKey.PublicKey()
	b.Signature = sig

	return nil
}

func (b *Block) Verify() error {
	if b.Signature == nil {
		return fmt.Errorf("block has no signature")
	}

	if !b.Signature.Verify(b.Validator, b.Header.Bytes()) {
		return fmt.Errorf("block has invalid signature")
	}

	for _, tx := range b.Transactions {
		if err := tx.Verify(); err != nil {
			return err
		}
	}

	dataHash, err := CalculateDataHash(b.Transactions)
	if err != nil {
		return err
	}

	if dataHash != b.DataHash {
		return fmt.Errorf("block (%s) has an invalid data hash", b.Hash(BlockHasher{}))
	}

	return nil
}

func (b *Block) Decode(dec Decoder[*Block]) error {
	return dec.Decode(b)
}

func (b *Block) Encode(enc Encoder[*Block]) error {
	return enc.Encode(b)
}

func (b *Block) Hash(hasher Hasher[*Header]) types.Hash {
	if b.hash.IsZero() {
		b.hash = hasher.Hash(b.Header)
	}

	return b.hash
}

func CalculateDataHash(txx []*Transaction) (hash types.Hash, err error) {
	buf := &bytes.Buffer{}

	for _, tx := range txx {
		if err = tx.Encode(NewGobTxEncoder(buf)); err != nil {
			return
		}
	}

	hash = sha256.Sum256(buf.Bytes())

	return
}
