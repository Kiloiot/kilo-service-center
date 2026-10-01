//go:build integration

package probe

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	mioty "github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	frameWriteTimeout = 5 * time.Second
	maxFramePayload   = 1 << 20
	frameIDLen        = len(mioty.MIOTYFrameIdentifier)
)

// writeFramePayload frames a JSON/MessagePack payload with the 8-byte
// protocol identifier and the little-endian payload size (BSSCI §3.1, SCACI
// §3.1).
func writeFramePayload(conn net.Conn, identifier [8]byte, payload []byte) error {
	frame := mioty.Frame{Identifier: identifier, Payload: payload}
	_ = conn.SetWriteDeadline(time.Now().Add(frameWriteTimeout))
	_, err := conn.Write(frame.Serialize())
	return err
}

// readFramePayload reads one frame and returns its payload.
func readFramePayload(conn net.Conn, identifier [8]byte) ([]byte, error) {
	header := make([]byte, mioty.FrameHeaderSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	if !bytes.Equal(header[:frameIDLen], identifier[:]) {
		return nil, fmt.Errorf("frame identifier %q, want %q", header[:frameIDLen], identifier[:])
	}
	size := binary.LittleEndian.Uint32(header[frameIDLen:])
	if size > maxFramePayload {
		return nil, fmt.Errorf("frame payload of %d bytes exceeds %d", size, maxFramePayload)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}
	return payload, nil
}
