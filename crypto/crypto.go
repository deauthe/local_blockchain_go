package crypto

import (
	"encoding/gob"
	"math/big"
)

func init() {
	gob.Register(&PrivateKey{})
	gob.Register(&PublicKey{})
	gob.Register(&Signature{})
	gob.Register(big.NewInt(0))
}

// ... rest of the file remains unchanged ...
