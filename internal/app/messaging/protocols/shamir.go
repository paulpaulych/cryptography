package protocols

import (
	"errors"
	"fmt"
	"github.com/paulpaulych/crypto/internal/app/algorithms/shamir"
	"github.com/paulpaulych/crypto/internal/app/messaging/msg-core"
	"github.com/paulpaulych/crypto/internal/app/messaging/nio"
	"github.com/paulpaulych/crypto/internal/app/tcp"
	"io"
	"log"
	. "math/big"
	. "net"
)

const blockSize = 4

func ShamirWriteFn(p *Int) func(msg io.Reader, conn Conn) error {
	return func(msg io.Reader, conn Conn) error {
		err := tcp.WriteBigIntWithLen(conn, p)
		if err != nil {
			errMsg := fmt.Sprintf("writing P failed: %v", err)
			return errors.New(errMsg)
		}

		err = nio.NewBlockTransfer(blockSize).WriteBlocks(nio.WriteProps{
			From:       msg,
			MetaWriter: conn,
			DataWriter: nio.NewFnWriter(shamirEncoder(p, conn)),
		})
		if err != nil {
			errMsg := fmt.Sprintf("error sending block: %v", err)
			return errors.New(errMsg)
		}
		return nil
	}
}

func shamirEncoder(p *Int, conn Conn) func([]byte) error {
	return func(block []byte) error {
		alice, err := shamir.InitAlice(p)
		if err != nil {
			log.Printf("failed to init alice: %d", err)
			return err
		}

		msgInt := new(Int).SetBytes(block)
		step1out, err := alice.Step1(msgInt)
		if err != nil {
			errMsg := fmt.Sprintf("writing step1out failed: %v", err)
			return errors.New(errMsg)
		}

		err = tcp.WriteBigIntWithLen(conn, step1out)
		if err != nil {
			errMsg := fmt.Sprintf("writing step1out failed: %v", err)
			return errors.New(errMsg)
		}

		step2out, err := tcp.ReadBigIntWithLen(conn)
		if err != nil {
			errMsg := fmt.Sprintf("reading step2out failed: %v", err)
			return errors.New(errMsg)
		}

		step3out := alice.Step3(step2out)
		err = tcp.WriteBigIntWithLen(conn, step3out)
		if err != nil {
			errMsg := fmt.Sprintf("writing step3out failed: %v", err)
			return errors.New(errMsg)
		}

		return nil
	}
}

func ShamirBob(
	output func(Addr) nio.ClosableWriter,
	onErr func(string),
) msg_core.Bob {
	return func(conn Conn) {
		out := output(conn.RemoteAddr())
		defer func() {
			err := out.Close()
			if err != nil {
				log.Printf("failed to close writer: %s", err)
			}
		}()

		p, err := tcp.ReadBigIntWithLen(conn)
		if err != nil {
			onErr(fmt.Sprintf("can't read p: %v", err))
			return
		}

		err = nio.NewBlockTransfer(blockSize).ReadBlocks(nio.ReadProps{
			MetaReader: conn,
			DataReader: nio.NewFnReader(decoder(p, conn)),
			To:         out,
		})
		if err != nil {
			onErr(fmt.Sprintf("can't transfer: %v", err))
			return
		}
	}
}

func decoder(p *Int, conn Conn) func(buf []byte) (int, error) {
	return func(buf []byte) (int, error) {
		bob, err := shamir.InitBob(p)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("failed to init bob: %d", err))
		}

		step1out, err := tcp.ReadBigIntWithLen(conn)
		if err == io.EOF {
			return 0, io.EOF
		}
		if err != nil {
			return 0, errors.New(fmt.Sprintf("can't read step1out: %v", err))
		}

		step2out := bob.Step2(step1out)
		err = tcp.WriteBigIntWithLen(conn, step2out)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("can't write step2out: %v", err))
		}

		step3out, err := tcp.ReadBigIntWithLen(conn)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("can't write step2out: %v", err))
		}
		bob.Decode(step3out).FillBytes(buf)
		return blockSize, nil
	}
}
