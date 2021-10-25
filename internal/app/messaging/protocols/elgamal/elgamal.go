package elgamal

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	. "math/big"
	. "net"

	msg_core "github.com/paulpaulych/crypto/internal/app/messaging/msg-core"
	"github.com/paulpaulych/crypto/internal/app/nio"
	dh "github.com/paulpaulych/crypto/internal/core/diffie-hellman"
	"github.com/paulpaulych/crypto/internal/core/elgamal-cipher"
	"github.com/paulpaulych/crypto/internal/core/rand"
)

const bobPubKeyFile = "bob_elgamal.key"

// TODO increase block size
const blockSize = 1

func NewConnWriteFn(
	commonPub dh.CommonPublicKey,
	bobPub *Int,
) msg_core.ConnWriteFn  {
	return func(msg io.Reader, conn Conn) error {
		err := nio.NewBlockTransfer(blockSize).WriteBlocks(nio.WriteProps{
			From:       msg,
			MetaWriter: conn,
			DataWriter: nio.NewFnWriter(encoder(commonPub, bobPub, conn)),
		})
		if err != nil {
			return fmt.Errorf("error sending block: %v", err)
		}
		return nil
	}
}

func encoder(
	commonPub dh.CommonPublicKey,
	bobPub *Int,
	conn Conn,
) func([]byte) error {
	return func(block []byte) error {
		alice := elgamal_cipher.NewAlice(commonPub, bobPub)
		fmt.Print(fmtElgamalAlice(alice))

		msgInt := new(Int).SetBytes(block)
		log.Printf("ELGAMAL: data int: %v", msgInt)
		encoded := alice.Encode(msgInt, rand.CryptoSafeRandom())
		log.Printf("ELGAMAL: R=%v, E=%v", encoded.R, encoded.E)
		err := nio.WriteBigIntWithLen(conn, encoded.R)
		if err != nil {
			return fmt.Errorf("writing R failed: %v", err)
		}
		err = nio.WriteBigIntWithLen(conn, encoded.E)
		if err != nil {
			return fmt.Errorf("writing E failed: %v", err)
		}

		return nil
	}
}

func ReadFn(
	cPub dh.CommonPublicKey,
	output func(Addr) nio.ClosableWriter,
) func(conn Conn) error {
	bob := elgamal_cipher.NewBob(cPub)
	fmt.Print(fmtElgamalBob(bob))

	return func(conn Conn) error {
		out := output(conn.RemoteAddr())
		defer func() {
			err := out.Close()
			if err != nil {
				log.Printf("failed to close writer: %s", err)
			}
		}()

		err := nio.NewBlockTransfer(blockSize).ReadBlocks(nio.ReadProps{
			MetaReader: conn,
			DataReader: nio.NewFnReader(decoder(bob, conn)),
			To:         out,
		})
		if err != nil {
			return fmt.Errorf("can't transfer: %v", err)
		}
		return nil
	}
}

func decoder(bob *elgamal_cipher.Bob, conn Conn) func(buf []byte) (int, error) {
	return func(buf []byte) (int, error) {
		R, err := nio.ReadBigIntWithLen(conn)
		if err != nil {
			return 0, fmt.Errorf("can't read R: %v", err)
		}

		E, err := nio.ReadBigIntWithLen(conn)
		if err != nil {
			return 0, fmt.Errorf("can't read E: %v", err)
		}
		encoded := &elgamal_cipher.Encoded{E: E, R: R}
		decoded := bob.Decode(encoded)
		if decoded.BitLen() > blockSize*8 {
			return 0, errors.New("received value is larger that buffer size. Seems like Alice uses incorrect key")
		}
		decoded.FillBytes(buf)
		log.Printf("ELGAMAL: decoded data=%v", buf)
		return blockSize, nil
	}
}

func fmtElgamalAlice(a *elgamal_cipher.Alice) string {
	return fmt.Sprintln("Elgamal node(Alice) initialized.\n",
		fmt.Sprintf("Common public key: P=%v, Q=%v\n", a.CommonPub.P(), a.CommonPub.G()),
		fmt.Sprintf("Bob public key: '%v'", a.BobPub),
	)
}

func fmtElgamalBob(bob *elgamal_cipher.Bob) string {
	err := ioutil.WriteFile(bobPubKeyFile, bob.Pub.Bytes(), 0644)
	if err != nil {
		return "error writing key"
	}
	return fmt.Sprintln("Elgamal node(Bob) initialized.\n",
		fmt.Sprintf("Common public key: P=%v, Q=%v\n", bob.CommonPub.P(), bob.CommonPub.G()),
		fmt.Sprintf("Node public key: '%v' (saved to %v)", bob.Pub, bobPubKeyFile),
	)
}
