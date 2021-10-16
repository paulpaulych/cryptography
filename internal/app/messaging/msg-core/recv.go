package msg_core

import (
	"github.com/paulpaulych/crypto/internal/app/tcp"
	"log"
	. "net"
)

type GetBobForProtocol = func(code ProtocolCode) (Bob, error)

func RecvMessage(getBob GetBobForProtocol) func(Conn) {
	return func(conn Conn) {
		defer func() { _ = conn.Close() }()

		protocolCode, err := tcp.ReadUint32(conn)
		if err != nil {
			log.Printf("failed to read protocol protocolCode: %s", err)
			return
		}
		read, err := getBob(protocolCode)
		if err != nil {
			log.Printf("bob error: %s", err)
			return
		}

		read(conn)
	}
}
