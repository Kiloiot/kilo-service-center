//go:build integration

package probe

import (
	"encoding/json"
	"math"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// Timing policy of the conformance suite.
const (
	frameWait   = 10 * time.Second // longest the SC may take to answer or emit a frame
	quietWindow = 3 * time.Second  // how long "the SC sends nothing" is observed
)

// wireFrame is one inbound frame exactly as the Service Center put it on the
// wire. fields comes from a generic decoder, so msgpack bin stays []byte and
// arrays stay []interface{}: tests can tell the two encodings apart.
type wireFrame struct {
	raw     []byte
	fields  map[string]interface{}
	command string
	opID    int64
}

func decodeWireFrame(payload []byte) (wireFrame, error) {
	f := wireFrame{raw: payload}
	var err error
	if len(payload) > 0 && payload[0] == '{' {
		err = json.Unmarshal(payload, &f.fields)
	} else {
		err = msgpack.Unmarshal(payload, &f.fields)
	}
	if err != nil {
		return f, err
	}
	f.command, _ = f.fields[keyCommand].(string)
	f.opID, _ = toInt64(f.fields[keyOpID])
	return f, nil
}

func (f wireFrame) has(key string) bool {
	_, ok := f.fields[key]
	return ok
}

func (f wireFrame) int(key string) (int64, bool) { return toInt64(f.fields[key]) }

func (f wireFrame) uint(key string) (uint64, bool) { return toUint64(f.fields[key]) }

func (f wireFrame) float(key string) (float64, bool) { return toFloat64(f.fields[key]) }

func (f wireFrame) bool(key string) (value, present bool) {
	value, present = f.fields[key].(bool)
	return value, present
}

// replyAction is how a simulated peer answers an operation the Service Center
// initiates.
type replyAction int

const (
	replyNormally replyAction = iota // the spec response
	replyError                       // an error frame with the rule's code
	replySilently                    // no answer at all
	replyDrop                        // close the connection instead of answering
)

type replyRule struct {
	action replyAction
	code   int
	match  func(wireFrame) bool // nil applies the rule to every frame of the command
	once   bool                 // the rule retires after its first use
}

// peer is one simulated BSSCI base station or SCACI application center. A
// reader goroutine records every inbound frame and answers what the Service
// Center initiates; tests send operations and await frames.
type peer struct {
	t          *testing.T
	name       string
	conn       net.Conn
	identifier [8]byte
	encodeJSON atomic.Bool
	// responseNames overrides the spec response of an SC-initiated command
	// (SCACI names the dlDataRes response txDataResRsp).
	responseNames map[string]string
	replyFields   map[string]func() map[string]interface{}

	writeMu sync.Mutex
	mu      sync.Mutex
	frames  []wireFrame
	arrived chan struct{}
	rules   map[string]replyRule
	nextOp  atomic.Int64
	done    chan struct{}
}

// newPeer wraps the connection; start launches the reader once the caller has
// set responseNames and replyFields, which the reader only reads.
func newPeer(t *testing.T, name string, conn net.Conn, identifier [8]byte) *peer {
	return &peer{
		t: t, name: name, conn: conn, identifier: identifier,
		responseNames: map[string]string{},
		replyFields:   map[string]func() map[string]interface{}{},
		arrived:       make(chan struct{}),
		rules:         map[string]replyRule{},
		done:          make(chan struct{}),
	}
}

func (p *peer) start() {
	go p.readLoop()
	p.t.Cleanup(p.close)
}

func (p *peer) close() {
	_ = p.conn.Close()
	<-p.done
}

func (p *peer) readLoop() {
	defer close(p.done)
	for {
		payload, err := readFramePayload(p.conn, p.identifier)
		if err != nil {
			return
		}
		f, err := decodeWireFrame(payload)
		if err != nil {
			p.t.Logf("%s: undecodable frame %x: %v", p.name, payload, err)
			continue
		}
		p.record(f)
		p.react(f)
	}
}

func (p *peer) record(f wireFrame) {
	p.mu.Lock()
	p.frames = append(p.frames, f)
	close(p.arrived)
	p.arrived = make(chan struct{})
	p.mu.Unlock()
}

// react answers the Service Center the way the spec requires: SC-initiated
// operations (negative opId) get their response, responses to our own
// operations get the complete, and error frames get errorAck.
func (p *peer) react(f wireFrame) {
	switch {
	case f.command == cmdError:
		p.writeQuiet(map[string]interface{}{keyCommand: cmdErrorAck, keyOpID: f.opID})
	case f.command == cmdErrorAck, strings.HasSuffix(f.command, suffixCmp):
	case strings.HasSuffix(f.command, suffixRsp):
		if f.opID > 0 {
			p.writeQuiet(map[string]interface{}{keyCommand: strings.TrimSuffix(f.command, suffixRsp) + suffixCmp, keyOpID: f.opID})
		}
	case f.opID < 0:
		p.answer(f)
	}
}

func (p *peer) answer(f wireFrame) {
	p.mu.Lock()
	rule, ok := p.rules[f.command]
	if ok && (rule.match == nil || rule.match(f)) {
		if rule.once {
			delete(p.rules, f.command)
		}
	} else {
		rule = replyRule{action: replyNormally}
	}
	p.mu.Unlock()
	switch rule.action {
	case replySilently:
	case replyDrop:
		dropAbortively(p.conn)
	case replyError:
		p.writeQuiet(map[string]interface{}{keyCommand: cmdError, keyOpID: f.opID, keyCode: rule.code, keyMessage: p.name})
	default:
		name, renamed := p.responseNames[f.command]
		if !renamed {
			name = f.command + suffixRsp
		}
		msg := map[string]interface{}{keyCommand: name, keyOpID: f.opID}
		if extra, has := p.replyFields[f.command]; has {
			for k, v := range extra() {
				msg[k] = v
			}
		}
		p.writeQuiet(msg)
	}
}

// onRequest sets how the peer answers an SC-initiated command.
func (p *peer) onRequest(command string, rule replyRule) {
	p.mu.Lock()
	p.rules[command] = rule
	p.mu.Unlock()
}

// mark returns the position after the last recorded frame; awaitFrom only
// looks at frames recorded from that position on.
func (p *peer) mark() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.frames)
}

func (p *peer) awaitFrom(from int, timeout time.Duration, match func(wireFrame) bool) (wireFrame, bool) {
	deadline := time.Now().Add(timeout)
	for {
		p.mu.Lock()
		for ; from < len(p.frames); from++ {
			if match(p.frames[from]) {
				f := p.frames[from]
				p.mu.Unlock()
				return f, true
			}
		}
		wait := p.arrived
		p.mu.Unlock()
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return wireFrame{}, false
		}
		timer := time.NewTimer(remaining)
		select {
		case <-wait:
			timer.Stop()
		case <-p.done:
			timer.Stop()
			if p.mark() == from {
				return wireFrame{}, false
			}
		case <-timer.C:
		}
	}
}

// framesFrom returns every frame recorded from position from on that matches.
func (p *peer) framesFrom(from int, match func(wireFrame) bool) []wireFrame {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []wireFrame
	for _, f := range p.frames[from:] {
		if match(f) {
			out = append(out, f)
		}
	}
	return out
}

// awaitCommand waits for a frame of the command whose fields satisfy match.
func (p *peer) awaitCommand(t *testing.T, from int, command string, match func(wireFrame) bool) wireFrame {
	t.Helper()
	f, ok := p.awaitFrom(from, frameWait, func(f wireFrame) bool {
		return f.command == command && (match == nil || match(f))
	})
	require.True(t, ok, "%s: no %s frame within %s", p.name, command, frameWait)
	return f
}

// send writes a peer-initiated message, assigning the next positive opId when
// the message carries none, and returns the opId.
func (p *peer) send(t *testing.T, msg map[string]interface{}) int64 {
	t.Helper()
	if _, ok := msg[keyOpID]; !ok {
		msg[keyOpID] = p.nextOp.Add(1)
	}
	require.NoError(t, p.write(msg), "%s: write %v", p.name, msg[keyCommand])
	op, _ := toInt64(msg[keyOpID])
	return op
}

// request sends a peer-initiated operation and returns the Service Center's
// answer to it: the operation's response or an error frame.
func (p *peer) request(t *testing.T, msg map[string]interface{}) wireFrame {
	t.Helper()
	from := p.mark()
	command, _ := msg[keyCommand].(string)
	op := p.send(t, msg)
	f, ok := p.awaitFrom(from, frameWait, func(f wireFrame) bool {
		return f.opID == op && (f.command == command+suffixRsp || f.command == cmdError)
	})
	require.True(t, ok, "%s: no answer to %s opId %d within %s", p.name, command, op, frameWait)
	return f
}

func (p *peer) write(msg map[string]interface{}) error {
	var (
		payload []byte
		err     error
	)
	if p.encodeJSON.Load() {
		payload, err = json.Marshal(msg)
	} else {
		payload, err = msgpack.Marshal(msg)
	}
	if err != nil {
		return err
	}
	return p.writePayload(payload)
}

func (p *peer) writePayload(payload []byte) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	return writeFramePayload(p.conn, p.identifier, payload)
}

// writeQuiet is used by the reader goroutine, where a write to a connection
// the test has already dropped is expected.
func (p *peer) writeQuiet(msg map[string]interface{}) {
	_ = p.write(msg)
}

// Numeric decoding across the encodings msgpack and JSON produce.

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		if n > math.MaxInt64 {
			return 0, false
		}
		return int64(n), true
	case float64:
		if n != math.Trunc(n) {
			return 0, false
		}
		return int64(n), true
	}
	return 0, false
}

func toUint64(v interface{}) (uint64, bool) {
	if n, ok := v.(uint64); ok {
		return n, true
	}
	n, ok := toInt64(v)
	if !ok || n < 0 {
		return 0, false
	}
	return uint64(n), true
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	i, ok := toInt64(v)
	return float64(i), ok
}

// numericArray reports whether v is encoded as a Numeric[n] array of bytes
// (an array of integers 0-255), and returns the bytes.
func numericArray(v interface{}) ([]byte, bool) {
	items, ok := v.([]interface{})
	if !ok {
		return nil, false
	}
	out := make([]byte, 0, len(items))
	for _, item := range items {
		n, isInt := toInt64(item)
		if !isInt || n < 0 || n > math.MaxUint8 {
			return nil, false
		}
		out = append(out, byte(n))
	}
	return out, true
}

// numeric encodes bytes in the spec's Numeric[n] form: an array of numbers.
func numeric(b []byte) []interface{} {
	out := make([]interface{}, len(b))
	for i, v := range b {
		out[i] = v
	}
	return out
}
