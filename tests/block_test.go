package core_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/deauthe/local_blockchain_go/core"
	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/deauthe/local_blockchain_go/types"
	"github.com/stretchr/testify/assert"
)

func TestSignBlock(t *testing.T) {
	privKey := crypto.GeneratePrivateKey()
	b := RandomBlock(t, 0, types.Hash{})

	assert.Nil(t, b.Sign(privKey))
	assert.NotNil(t, b.Signature)
}

func TestVerifyBlock(t *testing.T) {
	privKey := crypto.GeneratePrivateKey()
	b := RandomBlock(t, 0, types.Hash{})

	assert.Nil(t, b.Sign(privKey))
	assert.Nil(t, b.Verify())

	otherPrivKey := crypto.GeneratePrivateKey()
	b.Validator = otherPrivKey.PublicKey()
	assert.NotNil(t, b.Verify())

	b.Height = 100
	assert.NotNil(t, b.Verify())
}

func TestDecodeEncodeBlock(t *testing.T) {
	b := RandomBlock(t, 1, types.Hash{})
	buf := &bytes.Buffer{}
	assert.Nil(t, b.Encode(core.NewGobBlockEncoder(buf)))

	bDecode := new(core.Block)
	assert.Nil(t, bDecode.Decode(core.NewGobBlockDecoder(buf)))

	assert.Equal(t, b.Header, bDecode.Header)

	for i := 0; i < len(b.Transactions); i++ {
		b.Transactions[i].TxHash = types.Hash{}
		assert.Equal(t, b.Transactions[i], bDecode.Transactions[i])
	}

	assert.Equal(t, b.Validator, bDecode.Validator)
	assert.Equal(t, b.Signature, bDecode.Signature)
}

func RandomBlock(t *testing.T, height uint32, prevBlockHash types.Hash) *core.Block {
	privKey := crypto.GeneratePrivateKey()
	tx := randomTxWithSignature(t)
	header := &core.Header{
		Version:       1,
		PrevBlockHash: prevBlockHash,
		Height:        height,
		Timestamp:     time.Now().UnixNano(),
	}

	b, err := core.NewBlock(header, []*core.Transaction{tx})
	assert.Nil(t, err)
	dataHash, err := core.CalculateDataHash(b.Transactions)
	assert.Nil(t, err)
	b.Header.DataHash = dataHash
	assert.Nil(t, b.Sign(privKey))

	return b
}
