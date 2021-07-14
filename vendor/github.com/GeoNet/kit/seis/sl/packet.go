package sl

import (
	"fmt"
)

const (
	PacketSize = 8 + 512
)

type Packet struct {
	SL   [2]byte   // ASCII String == "SL"
	Seq  [6]byte   // ASCII sequence number
	Data [512]byte // Fixed size payload
}

type PacketError struct {
	message string
}

func NewPacketError(message string) *PacketError {
	return &PacketError{
		message: message,
	}
}

func (e *PacketError) Error() string {
	return e.message
}

func NewPacket(data []byte) (*Packet, error) {
	if l := len(data); l < PacketSize {
		return nil, NewPacketError(fmt.Sprintf("invalid packet data length: %d", l))
	}
	if data[0] != 'S' || data[1] != 'L' {
		return nil, NewPacketError(fmt.Sprintf("invalid packet header tag: %v", string(data[0:2])))
	}

	var pkt Packet

	copy(pkt.SL[:], data[0:2])
	copy(pkt.Seq[:], data[2:8])
	copy(pkt.Data[:], data[8:])

	return &pkt, nil
}
