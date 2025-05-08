package util

import (
	"github.com/deauthe/local_blockchain_go/core"
	"github.com/deauthe/local_blockchain_go/crypto"
)

// NewRandomTransaction creates a new random transaction for testing
func NewRandomTransaction(value uint64) *core.Transaction {
	privKey := crypto.GeneratePrivateKey()
	tx := core.NewTransaction(nil)
	tx.Value = value
	tx.From = privKey.PublicKey()
	tx.Sign(privKey)
	return tx
}
