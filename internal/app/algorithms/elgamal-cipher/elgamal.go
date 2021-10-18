package elgamal_cipher

import (
	"crypto/rand"
	. "github.com/paulpaulych/crypto/internal/app/algorithms/arythmetics"
	dh "github.com/paulpaulych/crypto/internal/app/algorithms/diffie-hellman"
	. "math/big"
)

type Encoded struct {
	R, E *Int
}

type Alice struct {
	CommonPub dh.CommonPublicKey
	BobPub    *Int
}

func NewAlice(commonPub dh.CommonPublicKey, bobPub *Int) *Alice {
	return &Alice{CommonPub: commonPub, BobPub: bobPub}
}

func (a Alice) Encode(msg *Int, rng func(max *Int) (*Int, error)) *Encoded {
	p := a.CommonPub.P()
	secret, err := rng(p)
	if err != nil {
		return nil
	}

	tmp := new(Int)
	tmp.Mul(msg, PowByMod(a.BobPub, secret, p))
	e := tmp.Mod(tmp, p)
	return &Encoded{
		R: PowByMod(a.CommonPub.G(), secret, p),
		E: e,
	}
}

type Bob struct {
	CommonPub dh.CommonPublicKey
	Pub       *Int
	sec       *Int
}

func NewBob(commonPub dh.CommonPublicKey) *Bob {
	maxSecret := new(Int).Sub(commonPub.P(), NewInt(1))
	secret, _ := rand.Int(rand.Reader, maxSecret)
	pub := PowByMod(commonPub.G(), secret, commonPub.P())
	return &Bob{
		CommonPub: commonPub,
		Pub:       pub,
		sec:       secret,
	}
}

func (b Bob) Decode(encoded *Encoded) *Int {
	p := b.CommonPub.P()
	tmp := new(Int)
	tmp.Sub(p, NewInt(1))
	tmp.Sub(tmp, b.sec)
	return tmp.
		Mul(encoded.E, PowByMod(encoded.R, tmp, p)).
		Mod(tmp, p)
}
