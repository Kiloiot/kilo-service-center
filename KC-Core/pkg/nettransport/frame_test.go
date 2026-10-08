package nettransport

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// Payload sizes exercising little-endian length encoding across byte boundaries.
const (
	framePayloadSizeOne   = 1
	framePayloadSize256   = 256
	framePayloadSize64Ki  = 65536
	testMaxPayload        = 1024 * 1024
	shortHeaderElevenByte = 11
	shortHeaderEightByte  = 8
	testWriteTimeout      = 50 * time.Millisecond
)

func scaciCodec() FrameCodec {
	return FrameCodec{identifier: mioty.SCACIFrameIdentifier, maxPayload: testMaxPayload, writeTimeout: testWriteTimeout}
}

func header(identifier [8]byte, size uint32) []byte {
	h := make([]byte, mioty.FrameHeaderSize)
	copy(h[:8], identifier[:])
	binary.LittleEndian.PutUint32(h[8:], size)
	return h
}

func TestFrameCodecParseHeader(t *testing.T) {
	codec := scaciCodec()

	t.Run("accepts the codec identifier", func(t *testing.T) {
		size, err := codec.ParseHeader(header(mioty.SCACIFrameIdentifier, 0))
		require.NoError(t, err)
		assert.Equal(t, uint32(0), size)
	})
	t.Run("rejects the other protocol identifier", func(t *testing.T) {
		_, err := codec.ParseHeader(header(mioty.MIOTYFrameIdentifier, 0))
		require.ErrorIs(t, err, ErrInvalidFrameIdentifier)
		assert.Contains(t, err.Error(), "MIOTYB01")
	})
	t.Run("rejects a lowercase identifier", func(t *testing.T) {
		var lower [8]byte
		copy(lower[:], "miotya01")
		_, err := codec.ParseHeader(header(lower, 0))
		require.ErrorIs(t, err, ErrInvalidFrameIdentifier)
	})
	t.Run("rejects a size over the cap", func(t *testing.T) {
		_, err := codec.ParseHeader(header(mioty.SCACIFrameIdentifier, testMaxPayload+1))
		require.ErrorIs(t, err, ErrPayloadTooLarge)
	})
	t.Run("accepts a size at the cap", func(t *testing.T) {
		size, err := codec.ParseHeader(header(mioty.SCACIFrameIdentifier, testMaxPayload))
		require.NoError(t, err)
		assert.Equal(t, uint32(testMaxPayload), size)
	})
	for _, n := range []int{shortHeaderElevenByte, shortHeaderEightByte, 0} {
		t.Run("rejects a short header", func(t *testing.T) {
			_, err := codec.ParseHeader(header(mioty.SCACIFrameIdentifier, 0)[:n])
			require.ErrorIs(t, err, ErrHeaderTooShort)
		})
	}
}

func TestFrameCodecEncodeLayout(t *testing.T) {
	codec := scaciCodec()
	cases := []struct {
		name string
		size int
		le   []byte
	}{
		{"one byte", framePayloadSizeOne, []byte{0x01, 0x00, 0x00, 0x00}},
		{"256 bytes", framePayloadSize256, []byte{0x00, 0x01, 0x00, 0x00}},
		{"64Ki bytes", framePayloadSize64Ki, []byte{0x00, 0x00, 0x01, 0x00}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := bytes.Repeat([]byte{0xAB}, tc.size)
			encoded, err := codec.Encode(payload)
			require.NoError(t, err)
			assert.Equal(t, []byte("MIOTYA01"), encoded[:8])
			assert.Equal(t, tc.le, encoded[8:mioty.FrameHeaderSize])
			assert.Equal(t, payload, encoded[mioty.FrameHeaderSize:])
		})
	}

	t.Run("empty payload is a bare header", func(t *testing.T) {
		encoded, err := codec.Encode(nil)
		require.NoError(t, err)
		assert.Len(t, encoded, mioty.FrameHeaderSize)
	})
	t.Run("payload over the cap is refused before any write", func(t *testing.T) {
		_, err := codec.Encode(make([]byte, testMaxPayload+1))
		require.ErrorIs(t, err, ErrPayloadTooLarge)
	})
}

func TestFrameCodecRead(t *testing.T) {
	codec := scaciCodec()

	t.Run("round trip", func(t *testing.T) {
		payload := []byte(`{"command":"connect","opId":0}`)
		encoded, err := codec.Encode(payload)
		require.NoError(t, err)
		frame, err := codec.Read(bytes.NewReader(encoded))
		require.NoError(t, err)
		assert.Equal(t, mioty.SCACIFrameIdentifier, frame.Identifier)
		assert.Equal(t, uint32(len(payload)), frame.PayloadSize)
		assert.Equal(t, payload, frame.Payload)
	})
	t.Run("empty payload reads with nil payload", func(t *testing.T) {
		frame, err := codec.Read(bytes.NewReader(header(mioty.SCACIFrameIdentifier, 0)))
		require.NoError(t, err)
		assert.Empty(t, frame.Payload)
	})
	t.Run("clean close before the header is io.EOF", func(t *testing.T) {
		_, err := codec.Read(bytes.NewReader(nil))
		assert.Equal(t, io.EOF, err)
	})
	t.Run("truncated header is not a clean close", func(t *testing.T) {
		_, err := codec.Read(bytes.NewReader(header(mioty.SCACIFrameIdentifier, 0)[:shortHeaderEightByte]))
		require.Error(t, err)
		assert.NotEqual(t, io.EOF, err)
	})
	t.Run("declared size beyond the data is a truncated frame", func(t *testing.T) {
		data := append(header(mioty.SCACIFrameIdentifier, 100), make([]byte, 50)...)
		_, err := codec.Read(bytes.NewReader(data))
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
	t.Run("wrong identifier is reported before the payload is read", func(t *testing.T) {
		data := append(header(mioty.MIOTYFrameIdentifier, 4), []byte("test")...)
		_, err := codec.Read(bytes.NewReader(data))
		require.ErrorIs(t, err, ErrInvalidFrameIdentifier)
	})
}

func TestFrameCodecWriteOverConnection(t *testing.T) {
	codec := scaciCodec()
	scConn, acConn := net.Pipe()
	defer func() { _ = scConn.Close() }()
	defer func() { _ = acConn.Close() }()

	payloads := [][]byte{
		[]byte(`{"command":"connectResponse","opId":0}`),
		bytes.Repeat([]byte{0x42}, framePayloadSize256),
		nil,
	}
	errCh := make(chan error, 1)
	go func() {
		for _, p := range payloads {
			encoded, err := codec.Encode(p)
			if err == nil {
				err = codec.Send(scConn, encoded)
			}
			if err != nil {
				errCh <- err
				return
			}
		}
		errCh <- nil
	}()

	for i, want := range payloads {
		frame, err := codec.Read(acConn)
		require.NoError(t, err, "frame %d", i)
		assert.Equal(t, uint32(len(want)), frame.PayloadSize, "frame %d", i)
		assert.Equal(t, want, frame.Payload, "frame %d", i)
	}
	require.NoError(t, <-errCh)
}

type shortWriter struct{ n int }

func (w shortWriter) Write(p []byte) (int, error) {
	if len(p) < w.n {
		return len(p), nil
	}
	return w.n, nil
}

func (shortWriter) SetWriteDeadline(time.Time) error { return nil }

// deadlineRecorder records every write deadline a Send sets.
type deadlineRecorder struct {
	deadlines []time.Time
	written   []byte
}

func (r *deadlineRecorder) Write(p []byte) (int, error) {
	r.written = append(r.written, p...)
	return len(p), nil
}

func (r *deadlineRecorder) SetWriteDeadline(t time.Time) error {
	r.deadlines = append(r.deadlines, t)
	return nil
}

func TestFrameCodecSendReportsShortWrite(t *testing.T) {
	codec := scaciCodec()
	encoded, err := codec.Encode([]byte("payload"))
	require.NoError(t, err)
	err = codec.Send(shortWriter{n: shortHeaderEightByte}, encoded)
	require.ErrorIs(t, err, ErrShortWrite)
}

// testStalledSendBound is how long a test waits before declaring a send blocked.
const testStalledSendBound = 2 * time.Second

// A peer that stops reading must not hold the sender past the write timeout.
func TestFrameCodecSendReturnsWithinWriteTimeoutOnStalledPeer(t *testing.T) {
	codec := scaciCodec()
	scConn, stalledPeer := net.Pipe()
	defer func() { _ = scConn.Close() }()
	defer func() { _ = stalledPeer.Close() }()

	encoded, err := codec.Encode([]byte("payload"))
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() { done <- codec.Send(scConn, encoded) }()

	select {
	case err := <-done:
		require.ErrorIs(t, err, os.ErrDeadlineExceeded)
	case <-time.After(testStalledSendBound):
		t.Fatal("Send blocked on a peer that stopped reading")
	}
}

func TestFrameCodecSendSetsThenClearsTheWriteDeadline(t *testing.T) {
	codec := scaciCodec()
	encoded, err := codec.Encode([]byte("payload"))
	require.NoError(t, err)

	recorder := &deadlineRecorder{}
	before := time.Now()
	require.NoError(t, codec.Send(recorder, encoded))

	require.Len(t, recorder.deadlines, 2)
	assert.False(t, recorder.deadlines[0].Before(before.Add(testWriteTimeout)), "the write is bounded by the codec timeout")
	assert.True(t, recorder.deadlines[1].IsZero(), "the deadline is cleared after the write")
	assert.Equal(t, encoded, recorder.written)
}

// A codec is only built with a positive write bound, so every Send is bounded.
func TestNewFrameCodecRefusesNonPositiveWriteTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{0, -testWriteTimeout} {
		_, err := NewFrameCodec(mioty.SCACIFrameIdentifier, testMaxPayload, timeout)
		require.ErrorIs(t, err, ErrWriteTimeoutRequired, "write timeout %s", timeout)
	}
	codec, err := NewFrameCodec(mioty.SCACIFrameIdentifier, testMaxPayload, testWriteTimeout)
	require.NoError(t, err)
	assert.Equal(t, scaciCodec(), codec)
}

func TestFrameCodecSendRefusesCodecWithoutWriteTimeout(t *testing.T) {
	codec := FrameCodec{identifier: mioty.SCACIFrameIdentifier, maxPayload: testMaxPayload}
	encoded, err := codec.Encode([]byte("payload"))
	require.NoError(t, err)

	recorder := &deadlineRecorder{}
	require.ErrorIs(t, codec.Send(recorder, encoded), ErrWriteTimeoutRequired)
	assert.Empty(t, recorder.written, "nothing reaches the wire without a write bound")
}

func TestFrameCodecBSSCIIdentifier(t *testing.T) {
	codec := FrameCodec{identifier: mioty.MIOTYFrameIdentifier, maxPayload: testMaxPayload}
	encoded, err := codec.Encode([]byte("x"))
	require.NoError(t, err)
	assert.Equal(t, []byte("MIOTYB01"), encoded[:8])
	_, err = codec.Read(bytes.NewReader(header(mioty.SCACIFrameIdentifier, 0)))
	require.ErrorIs(t, err, ErrInvalidFrameIdentifier)
}
