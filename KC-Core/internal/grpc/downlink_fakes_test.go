package grpc

import (
	"context"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/downlinks"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
)

// messageSvcFake is what the downlink listing, edit and DL RX status fakes
// of these tests provide.
type messageSvcFake interface {
	DownlinkQueueLister
	DownlinkResultsReader
	downlinks.PendingEditor
	DLRXStatusReader
	ServingStationLocator
}

// downlinkFakes are the collaborators a test's downlink handlers run on; a
// nil one is a fake that answers nothing in particular.
type downlinkFakes struct {
	messages  messageSvcFake
	queuer    downlinks.Queuer
	revoker   downlinks.Revoker
	endpoints downlinks.EndpointLookup
	events    downlinks.UpdateRecorder
	audit     AuditRecorder
}

// withDefaults fills the collaborators a test leaves unset.
func (f downlinkFakes) withDefaults() downlinkFakes {
	if f.messages == nil {
		f.messages = &fakeMessageSvc{}
	}
	if f.queuer == nil {
		f.queuer = &fakeSCACIQueuer{}
	}
	if f.revoker == nil {
		f.revoker = &fakeDownlinkRevoker{}
	}
	if f.endpoints == nil {
		f.endpoints = &fakeEndpointSvc{}
	}
	if f.events == nil {
		f.events = discardedDownlinkEvents{}
	}
	if f.audit == nil {
		f.audit = &captureAuditRecorder{}
	}
	return f
}

// discardedDownlinkEvents accepts every downlink announcement.
type discardedDownlinkEvents struct{}

func (discardedDownlinkEvents) RecordPendingUpdated(context.Context, *storage.DownlinkMessage) error {
	return nil
}

// downlinkHandlerDeps wires the downlink handler group onto fakes, the
// commands through the real downlink service.
func downlinkHandlerDeps(t testing.TB, f downlinkFakes) DownlinkHandlerDeps {
	t.Helper()
	deps, err := buildDownlinkHandlerDeps(f)
	if err != nil {
		t.Fatalf("downlink service: %v", err)
	}
	return deps
}

func buildDownlinkHandlerDeps(f downlinkFakes) (DownlinkHandlerDeps, error) {
	f = f.withDefaults()
	commands, err := downlinks.NewService(downlinks.Deps{
		Queuer: f.queuer, Editor: f.messages, Revoker: f.revoker, Endpoints: f.endpoints, Events: f.events, Audit: f.audit, Log: logger.NewNop(),
	})
	if err != nil {
		return DownlinkHandlerDeps{}, err
	}
	return DownlinkHandlerDeps{Commands: commands, Queue: f.messages, Results: f.messages}, nil
}

// testDownlinkHandlers builds the downlink handler group on fakes.
func testDownlinkHandlers(f downlinkFakes) *DownlinkHandlers {
	deps, err := buildDownlinkHandlerDeps(f)
	if err != nil {
		panic(err)
	}
	return &DownlinkHandlers{commands: deps.Commands, queue: deps.Queue, results: deps.Results, log: logger.NewNop()}
}

// useDownlinks rewires a test service's downlink handlers onto fakes.
func (s *CoreService) useDownlinks(f downlinkFakes) {
	s.DownlinkHandlers = testDownlinkHandlers(f)
}
