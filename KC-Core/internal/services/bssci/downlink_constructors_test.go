package bssciservices

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

func TestNewDownlinkDispatcher_RejectsMissingCollaborators(t *testing.T) {
	log := logger.NewNop()
	reserver := &mockReserverForDispatch{}
	queue := &mockMIOTYDownlinksForDispatch{}
	windows := &windowClaims{}
	send := (&mockSendFn{}).Send
	clk := testutil.NewFakeClock(dispatchTestNow)

	cases := map[string]struct {
		log      logger.Logger
		reserver DownlinkReserver
		queue    DownlinkConfirmer
		windows  DownlinkWindowClaimer
		send     SendDLQueueFunc
		clk      clock.Clock
		want     error
	}{
		"nil logger":        {reserver: reserver, queue: queue, windows: windows, send: send, clk: clk, want: ErrNilDispatcherLogger},
		"nil reserver":      {log: log, queue: queue, windows: windows, send: send, clk: clk, want: ErrNilDownlinkReserver},
		"nil confirmer":     {log: log, reserver: reserver, windows: windows, send: send, clk: clk, want: ErrNilDownlinkConfirmer},
		"nil window claims": {log: log, reserver: reserver, queue: queue, send: send, clk: clk, want: ErrNilDownlinkWindowClaimer},
		"nil send function": {log: log, reserver: reserver, queue: queue, windows: windows, clk: clk, want: ErrNilDownlinkSendFunc},
		"nil clock":         {log: log, reserver: reserver, queue: queue, windows: windows, send: send, want: ErrNilDispatcherClock},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dispatcher, err := NewDownlinkDispatcher(tc.log, tc.reserver, tc.queue, tc.windows, tc.send, tc.clk)
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, dispatcher)
		})
	}
}

// newRevokeAnswers builds the revoke answers a downlink service under test
// delegates to; ENOENT is the refusal code that says "not held".
func newRevokeAnswers(t *testing.T, log logger.Logger, tenants bssci.TenantResolver, revocations DownlinkRevocationWriter, expiries StationExpiryReporter) RevokeAnswerer {
	t.Helper()
	answers, err := NewRevokeAnswers(RevokeAnswerDeps{
		Logger: log, Tenants: tenants, Revocations: revocations, Expiries: expiries,
		Serializer: NewQueueSerializer(), NotHeldCodes: []int{bssci.POSIX_ENOENT},
	})
	require.NoError(t, err)
	return answers
}

func TestNewDownlinkService_RejectsMissingCollaborators(t *testing.T) {
	reporter := newReporterFixture(t).reporter
	complete := func() DownlinkServiceDeps {
		return DownlinkServiceDeps{
			Logger: logger.NewNop(), Tenants: NewTenantResolver(nil), Outcomes: &mockMIOTYDownlinksForDispatch{},
			Holders: &mockMIOTYDownlinksForDispatch{}, Results: reporter,
			Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
		}
	}
	revokes := newRevokeAnswers(t, logger.NewNop(), NewTenantResolver(nil), &mockMIOTYDownlinksForDispatch{}, reporter)
	cases := map[string]struct {
		unset   func(*DownlinkServiceDeps)
		revokes RevokeAnswerer
		want    error
	}{
		"nil logger":          {func(d *DownlinkServiceDeps) { d.Logger = nil }, revokes, ErrNilDownlinkServiceLogger},
		"nil tenant resolver": {func(d *DownlinkServiceDeps) { d.Tenants = nil }, revokes, ErrNilTenantResolver},
		"nil queue writer":    {func(d *DownlinkServiceDeps) { d.Outcomes = nil }, revokes, ErrNilDownlinkWriter},
		"nil holder writer":   {func(d *DownlinkServiceDeps) { d.Holders = nil }, revokes, ErrNilDownlinkHolderWriter},
		"nil result reporter": {func(d *DownlinkServiceDeps) { d.Results = nil }, revokes, ErrNilStationResultReporter},
		"nil serializer":      {func(d *DownlinkServiceDeps) { d.Serializer = nil }, revokes, ErrNilQueueSerializer},
		"nil clock":           {func(d *DownlinkServiceDeps) { d.Clock = nil }, revokes, ErrNilDownlinkServiceClock},
		"nil revoke answers":  {func(*DownlinkServiceDeps) {}, nil, ErrNilRevokeAnswerer},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			deps := complete()
			tc.unset(&deps)
			svc, err := NewDownlinkService(deps, tc.revokes)
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, svc)
		})
	}
}

func TestNewRevokeAnswers_RejectsMissingCollaborators(t *testing.T) {
	complete := func() RevokeAnswerDeps {
		return RevokeAnswerDeps{
			Logger: logger.NewNop(), Tenants: NewTenantResolver(nil), Revocations: &mockMIOTYDownlinksForDispatch{},
			Expiries: newReporterFixture(t).reporter, Serializer: NewQueueSerializer(), NotHeldCodes: []int{bssci.POSIX_ENOENT},
		}
	}
	cases := map[string]struct {
		unset func(*RevokeAnswerDeps)
		want  error
	}{
		"nil logger":            {func(d *RevokeAnswerDeps) { d.Logger = nil }, ErrNilDownlinkServiceLogger},
		"nil tenant resolver":   {func(d *RevokeAnswerDeps) { d.Tenants = nil }, ErrNilTenantResolver},
		"nil revocation writer": {func(d *RevokeAnswerDeps) { d.Revocations = nil }, ErrNilDownlinkRevocationWriter},
		"nil expiry reporter":   {func(d *RevokeAnswerDeps) { d.Expiries = nil }, ErrNilStationResultReporter},
		"nil serializer":        {func(d *RevokeAnswerDeps) { d.Serializer = nil }, ErrNilQueueSerializer},
		"no not-held code":      {func(d *RevokeAnswerDeps) { d.NotHeldCodes = nil }, ErrNoRevokeNotHeldCodes},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			deps := complete()
			tc.unset(&deps)
			answers, err := NewRevokeAnswers(deps)
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, answers)
		})
	}
}
