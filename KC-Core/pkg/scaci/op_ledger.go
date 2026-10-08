package scaci

import (
	"fmt"
	"sync"
)

// OpIDPair is a consistent snapshot of both operation ID counters (SCACI §3.2).
type OpIDPair struct {
	AC int64
	SC int64
}

// opLedger keeps a session's operation IDs (SCACI §3.2): the AC and SC
// counters, the AC operations a resume may reissue (§1) and the SC operations
// awaiting the application center's response. The connection handler,
// broadcasts and the idle monitor reach them only through its methods.
type opLedger struct {
	mu sync.Mutex
	// ac is the highest AC operation ID accepted, sc the last SC operation ID
	// issued; both start at initialOpIDCounter.
	ac int64
	sc int64
	// (reissueFloor, reissueCeiling] holds the AC operation IDs the
	// application center may reissue after a resume (SCACI §1, §3.2).
	reissueFloor   int64
	reissueCeiling int64
	// awaiting maps each SC operation awaiting the application center's
	// response to its request command.
	awaiting map[int64]string
}

// restore sets both counters to what a persisted session stored.
func (l *opLedger) restore(ids OpIDPair) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ac, l.sc = ids.AC, ids.SC
}

// pair returns a consistent snapshot of both counters.
func (l *opLedger) pair() OpIDPair {
	l.mu.Lock()
	defer l.mu.Unlock()
	return OpIDPair{AC: l.ac, SC: l.sc}
}

// nextSC reserves the next SC operation ID: negative and strictly
// decrementing (SCACI §3.2).
func (l *opLedger) nextSC() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sc--
	return l.sc
}

// continueFrom keeps the SC counter at or below issued, the last SC
// operation ID a previous holder of the session issued (SCACI §3.2).
func (l *opLedger) continueFrom(issued int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sc = min(l.sc, issued)
}

// await registers an SC operation whose request is about to be written.
func (l *opLedger) await(opId int64, command string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.awaiting == nil {
		l.awaiting = make(map[int64]string)
	}
	l.awaiting[opId] = command
}

// settle consumes the SC operation a response answers; it reports false when
// no operation with that opId and request command awaits one.
func (l *opLedger) settle(opId int64, command string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.awaiting[opId] != command {
		return false
	}
	delete(l.awaiting, opId)
	return true
}

// openReissueWindow admits the AC operations above snAcOpId that the service
// center already knows (SCACI §1, §3.2, §3.3.1).
func (l *opLedger) openReissueWindow(snAcOpId int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reissueFloor = max(snAcOpId, initialOpIDCounter)
	l.reissueCeiling = max(l.ac, l.reissueFloor)
}

// acceptAC records the opId of an operation the application center starts: a
// new one above the counter closes the reissue window, a reissued one must be
// the next in the window (SCACI §1, §3.2).
func (l *opLedger) acceptAC(opId int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if opId > l.ac {
		l.ac = opId
		l.reissueFloor = l.reissueCeiling
		return nil
	}
	if l.reissueFloor < opId && opId <= l.reissueCeiling {
		l.reissueFloor = opId
		return nil
	}
	return fmt.Errorf(errFmtACOpIDMustIncrement, l.ac, opId)
}

// ResumeOpIDConflict explains why the application center's view of the
// operation IDs contradicts the counters the service center stored for the
// session, or returns "" when both agree (SCACI §3.3.1): snAcOpId is the
// minimum AC operation ID the service center must know, snScOpId the maximum
// SC operation ID the application center knows. The operations beyond those
// bounds are the ones a resume reissues (§1).
func ResumeOpIDConflict(snAcOpID, snScOpID int64, stored OpIDPair) string {
	if snAcOpID < 0 || snAcOpID > stored.AC {
		return fmt.Sprintf(resumeReasonFmtACOpIDUnknown, snAcOpID, stored.AC)
	}
	if snScOpID > 0 || snScOpID < stored.SC {
		return fmt.Sprintf(resumeReasonFmtSCOpIDNeverIssued, snScOpID, stored.SC)
	}
	return ""
}

//revive:disable:var-naming the Session methods keep the opId spelling of MIOTY SCACI §3.2

// NextScOpId reserves the next Service Center operation ID (SCACI §3.2).
func (s *Session) NextScOpId() int64 {
	return s.ops.nextSC()
}

// AwaitScResponse registers an SC operation whose request is about to be
// written, so its response is recognized (SCACI §3.2).
func (s *Session) AwaitScResponse(opId int64, command string) {
	s.ops.await(opId, command)
}

// SettleScResponse consumes the SC operation a response answers. It reports
// false when no operation with that opId and request command awaits one.
func (s *Session) SettleScResponse(opId int64, command string) bool {
	return s.ops.settle(opId, command)
}

// OpIDs returns a consistent snapshot of both operation ID counters.
func (s *Session) OpIDs() OpIDPair {
	return s.ops.pair()
}

// OpenReissueWindow admits, after a resume, the AC operations above
// snAcOpId that the service center already knows: the application center
// reissues its uncompleted operations with their original IDs (SCACI §1,
// §3.2), and snAcOpId is the minimum AC ID it requires the service center to
// know (§3.3.1).
func (s *Session) OpenReissueWindow(snAcOpId int64) {
	s.ops.openReissueWindow(snAcOpId)
}

// AcceptAcOpId validates and records the opId of an operation the
// application center starts. SCACI §3.2: AC operations use positive, strictly
// incrementing IDs; after a resume the application center first reissues its
// uncompleted operations, in order and with their original IDs, from the
// reissue window. A new operation closes the window.
func (s *Session) AcceptAcOpId(opId int64) error {
	if opId == OpIDConnect {
		return errOpIDReservedForConnect
	}
	if opId < 0 {
		return fmt.Errorf(errFmtACOpIDMustBePositive, opId)
	}
	return s.ops.acceptAC(opId)
}

// continueFrom carries on the SC operation IDs of prev, the held session this
// one resumes: they keep decrementing from the last one issued (SCACI §3.2).
func (s *Session) continueFrom(prev *Session) {
	s.ops.continueFrom(prev.OpIDs().SC)
}
