package bssciservices

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestNewDownlinkService_RejectsMissingCollaborators(t *testing.T) {
	complete := func() DownlinkServiceDeps {
		return DownlinkServiceDeps{
			Logger: logger.NewNop(), Tenants: NewTenantResolver(nil), Outcomes: &mockMIOTYDownlinksForDispatch{},
			Holders: &mockMIOTYDownlinksForDispatch{}, Results: newReporterFixture(t).reporter,
			Serializer: NewQueueSerializer(), Clock: testutil.NewFakeClock(dispatchTestNow),
		}
	}
	cases := map[string]struct {
		unset func(*DownlinkServiceDeps)
		want  error
	}{
		"nil logger":          {func(d *DownlinkServiceDeps) { d.Logger = nil }, ErrNilDownlinkServiceLogger},
		"nil tenant resolver": {func(d *DownlinkServiceDeps) { d.Tenants = nil }, ErrNilTenantResolver},
		"nil queue writer":    {func(d *DownlinkServiceDeps) { d.Outcomes = nil }, ErrNilDownlinkWriter},
		"nil holder writer":   {func(d *DownlinkServiceDeps) { d.Holders = nil }, ErrNilDownlinkHolderWriter},
		"nil result reporter": {func(d *DownlinkServiceDeps) { d.Results = nil }, ErrNilStationResultReporter},
		"nil serializer":      {func(d *DownlinkServiceDeps) { d.Serializer = nil }, ErrNilQueueSerializer},
		"nil clock":           {func(d *DownlinkServiceDeps) { d.Clock = nil }, ErrNilDownlinkServiceClock},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			deps := complete()
			tc.unset(&deps)
			svc, err := NewDownlinkService(deps)
			require.ErrorIs(t, err, tc.want)
			assert.Nil(t, svc)
		})
	}
}
