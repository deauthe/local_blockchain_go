package types

import (
	"encoding/gob"
	"encoding/hex"
	"fmt"
)

type Hash [32]byte

func init() {
	gob.Register(Hash{})
}

func (h Hash) IsZero() bool {
	for i := 0; i < 32; i++ {
		if h[i] != 0 {
			return false
		}
	}
	return true
}

func (h Hash) ToSlice() []byte {
	b := make([]byte, 32)
	for i := 0; i < 32; i++ {
		b[i] = h[i]
	}
	return b
}

func (h Hash) String() string {
	return hex.EncodeToString(h.ToSlice())
}

func HashFromBytes(b []byte) Hash {
	if len(b) != 32 {
		panic(fmt.Sprintf("invalid hash length: %d", len(b)))
	}
	var h Hash
	copy(h[:], b)
	return h
}

func HashFromString(s string) Hash {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(fmt.Sprintf("invalid hash string: %s", s))
	}
	return HashFromBytes(b)
}
