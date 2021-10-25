package protocols

import (
	"fmt"
	"github.com/paulpaulych/crypto/internal/app/messaging/msg-core"
	"github.com/paulpaulych/crypto/internal/app/messaging/protocols/elgamal"
	"github.com/paulpaulych/crypto/internal/app/messaging/protocols/rsa"
	"github.com/paulpaulych/crypto/internal/app/messaging/protocols/shamir"
	"github.com/paulpaulych/crypto/internal/app/nio"
	dh "github.com/paulpaulych/crypto/internal/core/diffie-hellman"
	"io/ioutil"
	"math/big"
	. "net"
)

const (
	Shamir msg_core.ProtocolCode = iota
	Elgamal
	Rsa
)

func ShamirWriter(p *big.Int) (msg_core.ConnWriter, error) {
	write, err := shamir.NewConnWriteFn(p)
	if err != nil {
		return nil, err
	}
	return msg_core.NewConnWriter(Shamir, write), nil
}

func ShamirReader(out func(addr Addr) nio.ClosableWriter) msg_core.ConnReader {
	return msg_core.NewConnReader(Shamir, shamir.ReadFn(out))
}

func ElgamalWriter(cPub dh.CommonPublicKey, bobPubFileName string) (msg_core.ConnWriter, error) {
	bytes, err := ioutil.ReadFile(bobPubFileName)
	if err != nil {
		return nil, err
	}
	bobPub := new(big.Int).SetBytes(bytes)
	writeFn := elgamal.NewConnWriteFn(cPub, bobPub)
	return msg_core.NewConnWriter(Elgamal, writeFn), nil
}

func ElgamalReader(p, g *big.Int, out func(addr Addr) nio.ClosableWriter) (msg_core.ConnReader, error) {
	commonPub, e := dh.NewCommonPublicKey(p, g)
	if e != nil {
		return nil, fmt.Errorf("Diffie-Hellman public key error: %v", e)
	}
	return msg_core.NewConnReader(Elgamal, elgamal.ReadFn(commonPub, out)), nil
}

func RsaWriter(bobPubFileName string) msg_core.ConnWriter {
	return msg_core.NewConnWriter(Rsa, rsa.WriteFn(bobPubFileName))
}

func RsaReader(p, q *big.Int, out func(addr Addr) nio.ClosableWriter) msg_core.ConnReader {
	return msg_core.NewConnReader(Rsa, rsa.ReadFn(p, q, out))
}
