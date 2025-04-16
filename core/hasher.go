package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"

	"github.com/deauthe/local_blockchain_go/types"
)

type Hasher[T any] interface {
	Hash(T) types.Hash
}

type BlockHasher struct{}

func (BlockHasher) Hash(b *Header) types.Hash {
	h := sha256.Sum256(b.Bytes())
	return types.Hash(h)
}

type TxHasher struct{}

// Hash will hash the whole bytes of the TX including the TxInner field
func (TxHasher) Hash(tx *Transaction) types.Hash {
	buf := new(bytes.Buffer)

	// Encode TxInner if it exists
	if tx.TxInner != nil {
		if err := gob.NewEncoder(buf).Encode(tx.TxInner); err != nil {
			panic(err)
		}
	}

	binary.Write(buf, binary.LittleEndian, tx.Data)
	binary.Write(buf, binary.LittleEndian, tx.To)
	binary.Write(buf, binary.LittleEndian, tx.Value)
	binary.Write(buf, binary.LittleEndian, tx.From)
	binary.Write(buf, binary.LittleEndian, tx.Nonce)

	return types.Hash(sha256.Sum256(buf.Bytes()))
}
