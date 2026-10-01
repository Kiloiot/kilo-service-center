package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// ackResultsMessageSvc serves one acknowledged and one unacknowledged downlink result.
type ackResultsMessageSvc struct {
	fakeMessageSvc
	ackedAt time.Time
}

func (m *ackResultsMessageSvc) GetDownlinkResults(context.Context, int64, *uuid.UUID, storage.DownlinkResultFilter, int, int) ([]*storage.DownlinkMessage, int, error) {
	return []*storage.DownlinkMessage{
		{ID: 1, QueID: 11, Status: mioty.DLQueueStatusTransmitted, Result: mioty.ResultSent, EndpointAckedAt: &m.ackedAt},
		{ID: 2, QueID: 12, Status: mioty.DLQueueStatusTransmitted, Result: mioty.ResultSent},
	}, 2, nil
}

// TestGetDownlinkResults_ReportsTheEndpointAcknowledgement: a downlink the
// endpoint acknowledged with dlAck (BSSCI §3.10.1) carries the time of that
// acknowledgement; one it did not acknowledge carries none.
func TestGetDownlinkResults_ReportsTheEndpointAcknowledgement(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})
	ackedAt := time.Date(2026, 9, 27, 12, 30, 0, 0, time.UTC)
	svc.useDownlinks(downlinkFakes{messages: &ackResultsMessageSvc{ackedAt: ackedAt}})

	resp, err := svc.GetDownlinkResults(ownerCtx(), &pb.GetDownlinkResultsRequest{})

	require.NoError(t, err)
	require.Len(t, resp.Results, 2)
	require.NotNil(t, resp.Results[0].EndpointAckedAt)
	assert.True(t, ackedAt.Equal(resp.Results[0].EndpointAckedAt.AsTime()))
	assert.Nil(t, resp.Results[1].EndpointAckedAt)
}

// acceptedQueueMessageSvc serves one queued downlink a station accepted and one it has not.
type acceptedQueueMessageSvc struct {
	fakeMessageSvc
	acceptedAt time.Time
}

func (m *acceptedQueueMessageSvc) ListDownlinkQueue(context.Context, int64, storage.DownlinkQueueFilter, int, int) ([]*storage.DownlinkMessage, int64, error) {
	return []*storage.DownlinkMessage{
		{QueID: 21, Status: mioty.DLQueueStatusQueued, BsEui: 0x70B3D59CD00009E6, AcceptedAt: &m.acceptedAt},
		{QueID: 22, Status: mioty.DLQueueStatusQueued, BsEui: 0x70B3D59CD00009E6},
	}, 2, nil
}

// TestListDownlinkQueue_ReportsTheStationThatAcceptedADownlink: a queued
// downlink names the station holding it and when that station accepted it
// (dlDataQueRsp, BSSCI §3.12); one not yet accepted carries no time.
func TestListDownlinkQueue_ReportsTheStationThatAcceptedADownlink(t *testing.T) {
	svc := newRetainedService(t, &retainedFakes{})
	acceptedAt := time.Date(2026, 9, 28, 12, 50, 0, 0, time.UTC)
	svc.useDownlinks(downlinkFakes{messages: &acceptedQueueMessageSvc{acceptedAt: acceptedAt}})

	resp, err := svc.ListDownlinkQueue(ownerCtx(), &pb.ListDownlinkQueueRequest{})

	require.NoError(t, err)
	require.Len(t, resp.Messages, 2)
	assert.Equal(t, "70B3D59CD00009E6", resp.Messages[0].BsEui)
	require.NotNil(t, resp.Messages[0].AcceptedAt)
	assert.True(t, acceptedAt.Equal(resp.Messages[0].AcceptedAt.AsTime()))
	assert.Nil(t, resp.Messages[1].AcceptedAt)
}
