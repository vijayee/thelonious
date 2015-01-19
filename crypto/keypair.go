package crypto

import (
	"strings"

	"github.com/eris-ltd/new-thelonious/thelutil"
	"github.com/obscuren/secp256k1-go"
)

type KeyPair struct {
	PrivateKey []byte
	PublicKey  []byte
	address    []byte
	mnemonic   string
	// The associated account
	// account *StateObject
}

func GenerateNewKeyPair() *KeyPair {
	_, prv := secp256k1.GenerateKeyPair()
	keyPair, _ := NewKeyPairFromSec(prv) // swallow error, this one cannot err
	return keyPair
}

func NewKeyPairFromSec(seckey []byte) (*KeyPair, error) {
	pubkey, err := secp256k1.GeneratePubKey(seckey)
	if err != nil {
		return nil, err
	}

	return &KeyPair{PrivateKey: seckey, PublicKey: pubkey}, nil
}

func (k *KeyPair) Address() []byte {
	if k.address == nil {
		k.address = Sha3(k.PublicKey[1:])[12:]
	}
	return k.address
}

func (k *KeyPair) Mnemonic() string {
	if k.mnemonic == "" {
		k.mnemonic = strings.Join(MnemonicEncode(thelutil.Bytes2Hex(k.PrivateKey)), " ")
	}
	return k.mnemonic
}

func (k *KeyPair) AsStrings() (string, string, string, string) {
	return k.Mnemonic(), thelutil.Bytes2Hex(k.Address()), thelutil.Bytes2Hex(k.PrivateKey), thelutil.Bytes2Hex(k.PublicKey)
}

func (k *KeyPair) RlpEncode() []byte {
	return k.RlpValue().Encode()
}

func (k *KeyPair) RlpValue() *thelutil.Value {
	return thelutil.NewValue(k.PrivateKey)
}
