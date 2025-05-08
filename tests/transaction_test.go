package core_test

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/deauthe/local_blockchain_go/core"
	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/deauthe/local_blockchain_go/types"
	"github.com/stretchr/testify/assert"
)

func TestVerifyTransactionWithTamper(t *testing.T) {
	tx := core.NewTransaction(nil)

	fromPrivKey := crypto.GeneratePrivateKey()
	toPrivKey := crypto.GeneratePrivateKey()
	hackerPrivKey := crypto.GeneratePrivateKey()

	tx.From = fromPrivKey.PublicKey()
	tx.To = toPrivKey.PublicKey()
	tx.Value = 666

	assert.Nil(t, tx.Sign(fromPrivKey))
	tx.TxHash = types.Hash{}

	tx.To = hackerPrivKey.PublicKey()

	assert.NotNil(t, tx.Verify())
}

func TestNFTTransaction(t *testing.T) {
	collectionTx := core.CollectionTx{
		Fee:      200,
		MetaData: []byte("The beginning of a new collection"),
	}

	privKey := crypto.GeneratePrivateKey()
	tx := &core.Transaction{
		TxInner: collectionTx,
	}
	tx.Sign(privKey)
	tx.TxHash = types.Hash{}

	buf := new(bytes.Buffer)
	assert.Nil(t, gob.NewEncoder(buf).Encode(tx))

	txDecoded := &core.Transaction{}
	assert.Nil(t, gob.NewDecoder(buf).Decode(txDecoded))
	assert.Equal(t, tx, txDecoded)
}

func TestNativeTransferTransaction(t *testing.T) {
	fromPrivKey := crypto.GeneratePrivateKey()
	toPrivKey := crypto.GeneratePrivateKey()
	tx := &core.Transaction{
		To:    toPrivKey.PublicKey(),
		Value: 666,
	}

	assert.Nil(t, tx.Sign(fromPrivKey))
}

func TestSignTransaction(t *testing.T) {
	privKey := crypto.GeneratePrivateKey()
	tx := &core.Transaction{
		Data: []byte("foo"),
	}

	assert.Nil(t, tx.Sign(privKey))
	assert.NotNil(t, tx.Signature)
}

func TestVerifyTransaction(t *testing.T) {
	privKey := crypto.GeneratePrivateKey()
	tx := &core.Transaction{
		Data: []byte("foo"),
	}

	assert.Nil(t, tx.Sign(privKey))
	assert.Nil(t, tx.Verify())

	otherPrivKey := crypto.GeneratePrivateKey()
	tx.From = otherPrivKey.PublicKey()

	assert.NotNil(t, tx.Verify())
}

func TestTxEncodeDecode(t *testing.T) {
	tx := randomTxWithSignature(t)
	buf := &bytes.Buffer{}
	assert.Nil(t, tx.Encode(core.NewGobTxEncoder(buf)))
	tx.TxHash = types.Hash{}

	txDecoded := new(core.Transaction)
	assert.Nil(t, txDecoded.Decode(core.NewGobTxDecoder(buf)))
	assert.Equal(t, tx, txDecoded)
}

func randomTxWithSignature(t *testing.T) *core.Transaction {
	privKey := crypto.GeneratePrivateKey()
	tx := core.Transaction{
		Data: []byte("foo"),
	}
	assert.Nil(t, tx.Sign(privKey))

	return &tx
}
