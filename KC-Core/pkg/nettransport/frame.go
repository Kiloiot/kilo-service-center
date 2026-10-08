package nettransport

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// FrameCodec reads and writes the length-prefixed frames both MIOTY
// interfaces use: an 8-byte ASCII identifier, a little-endian uint32 payload
// size and the encoded message (BSSCI §2.1, SCACI §3.1).
type FrameCodec struct {
	identifier [8]byte
	maxPayload uint32
	// writeTimeout bounds every Send so a peer that stops reading cannot
	// hold the sender; a codec without one refuses to send.
	writeTimeout time.Duration
}

// NewFrameCodec builds the codec of one interface; a write timeout that is not
// positive would leave Send unbounded or failing at once, so it is refused.
func NewFrameCodec(identifier [8]byte, maxPayload uint32, writeTimeout time.Duration) (FrameCodec, error) {
	if writeTimeout <= 0 {
		return FrameCodec{}, fmt.Errorf(errFmtWriteTimeoutNotPositive, ErrWriteTimeoutRequired, writeTimeout)
	}
	return FrameCodec{identifier: identifier, maxPayload: maxPayload, writeTimeout: writeTimeout}, nil
}

// DeadlineWriter is the connection surface a frame write needs.
type DeadlineWriter interface {
	io.Writer
	SetWriteDeadline(t time.Time) error
}

// ParseHeader validates a frame header and returns the payload size it declares.
func (c FrameCodec) ParseHeader(header []byte) (uint32, error) {
	if len(header) < mioty.FrameHeaderSize {
		return 0, fmt.Errorf(errFmtHeaderTooShort, ErrHeaderTooShort, mioty.FrameHeaderSize, len(header))
	}
	var identifier [8]byte
	copy(identifier[:], header[:8])
	if identifier != c.identifier {
		return 0, fmt.Errorf(errFmtInvalidIdentifier, ErrInvalidFrameIdentifier, string(c.identifier[:]), string(identifier[:]))
	}
	size := binary.LittleEndian.Uint32(header[8:mioty.FrameHeaderSize])
	if size > c.maxPayload {
		return 0, fmt.Errorf(errFmtPayloadTooLarge, ErrPayloadTooLarge, size, c.maxPayload)
	}
	return size, nil
}

// Read consumes exactly one frame. A connection closed before the first
// header byte surfaces as io.EOF so callers can tell a clean close from a
// truncated frame.
func (c FrameCodec) Read(r io.Reader) (*mioty.Frame, error) {
	header := make([]byte, mioty.FrameHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	size, err := c.ParseHeader(header)
	if err != nil {
		return nil, err
	}
	frame := &mioty.Frame{Identifier: c.identifier, PayloadSize: size}
	if size > 0 {
		frame.Payload = make([]byte, size)
		if _, err := io.ReadFull(r, frame.Payload); err != nil {
			return nil, fmt.Errorf(errFmtReadPayload, err)
		}
	}
	return frame, nil
}

// Encode renders payload as one wire frame.
func (c FrameCodec) Encode(payload []byte) ([]byte, error) {
	if uint64(len(payload)) > uint64(c.maxPayload) {
		return nil, fmt.Errorf(errFmtPayloadTooLarge, ErrPayloadTooLarge, len(payload), c.maxPayload)
	}
	frame := mioty.Frame{Identifier: c.identifier, PayloadSize: uint32(len(payload)), Payload: payload} //nolint:gosec // bounded by maxPayload above
	return frame.Serialize(), nil
}

// Send writes an encoded frame in a single write bounded by the write timeout; a
// stalled peer surfaces as os.ErrDeadlineExceeded. A failed write may leave
// part of the frame on the wire, so callers close the connection. The
// deadline is cleared afterwards so it cannot fail writes the TLS layer
// issues while reading.
func (c FrameCodec) Send(w DeadlineWriter, encoded []byte) error {
	if c.writeTimeout <= 0 {
		return ErrWriteTimeoutRequired
	}
	// Socket deadlines are wall-clock instants checked by the runtime poller.
	if err := w.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		return fmt.Errorf(errFmtSetWriteDeadline, err)
	}
	n, err := w.Write(encoded)
	clearErr := w.SetWriteDeadline(time.Time{})
	if err != nil {
		return err
	}
	if n < len(encoded) {
		return fmt.Errorf(errFmtShortWrite, ErrShortWrite, n, len(encoded))
	}
	if clearErr != nil {
		return fmt.Errorf(errFmtClearWriteDeadline, clearErr)
	}
	return nil
}
