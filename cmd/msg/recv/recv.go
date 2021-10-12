package recv

import (
	"fmt"
	"github.com/paulpaulych/crypto/internal/app/messaging/msg-core"
	"github.com/paulpaulych/crypto/internal/app/messaging/nio"
	"github.com/paulpaulych/crypto/internal/app/messaging/protocols"
	"github.com/paulpaulych/crypto/internal/app/tcp"
	"github.com/paulpaulych/crypto/internal/infra/cli"
	"log"
	"net"
	"os"
)

type RecvConf struct{}

func (conf *RecvConf) CmdName() string {
	return "recv"
}

func (conf *RecvConf) InitCmd(args []string) (cli.Cmd, cli.CmdConfError) {
	flagsSpec := cli.NewFlagSpec(conf.CmdName(), map[string]string{
		"host": "host to bind",
		"port": "port to bind",
	})

	flags, err := flagsSpec.Parse(args)
	if err != nil {
		return nil, err
	}

	host := flags.Flags["host"].GetOr("localhost")
	port := flags.Flags["port"].GetOr("4444")
	addr := net.JoinHostPort(host, port)

	if err != nil {
		return nil, err
	}
	return &RecvCmd{bindAddr: addr}, nil
}

type RecvCmd struct {
	bindAddr string
}

func (cmd *RecvCmd) Run() error {
	return tcp.StartServer(cmd.bindAddr, msg_core.RecvMessage(chooseBob))
}

func chooseBob(code msg_core.ProtocolCode) (msg_core.Bob, error) {
	onErr := func(e string) {
		log.Printf("error reading message: %s", e)
	}
	newWriter := func(from net.Addr) nio.BlockWriter {
		return &consoleWriter{addr: from.String(), isFirst: true}
	}
	return protocols.ChooseBob(code, newWriter, onErr)
}

type consoleWriter struct {
	addr    string
	isFirst bool
}

func (w *consoleWriter) Write(p []byte, hasMore bool) error {
	if w.isFirst {
		fmt.Printf("RECEIVED MESSAGE FROM %s: ", w.addr)
		_, err := os.Stdout.Write(p)
		if err != nil {
			return err
		}
		if !hasMore {
			fmt.Println()
		}
		w.isFirst = false
	}
	_, err := os.Stdout.Write(p)
	return err
}
