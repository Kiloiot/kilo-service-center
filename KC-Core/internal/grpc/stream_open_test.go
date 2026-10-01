package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const (
	streamedHeader  = "header"
	streamedMessage = "message"
)

// recordingStream records the order of the response header and the messages it is sent.
type recordingStream[T any] struct {
	grpc.ServerStreamingServer[T]
	ctx   context.Context
	calls *[]string
}

func (s recordingStream[T]) Context() context.Context { return s.ctx }

func (s recordingStream[T]) SendHeader(metadata.MD) error {
	*s.calls = append(*s.calls, streamedHeader)
	return nil
}

func (s recordingStream[T]) Send(*T) error {
	*s.calls = append(*s.calls, streamedMessage)
	return nil
}

func newRecordingStream[T any](ctx context.Context) (recordingStream[T], *[]string) {
	calls := &[]string{}
	return recordingStream[T]{ctx: ctx, calls: calls}, calls
}

// quietStationListing streams no base station uplink, as a station with no traffic.
type quietStationListing struct {
	retainedMessageListing
}

func (quietStationListing) StreamBaseStationMessages(context.Context, int64, []byte, *grpcservices.MessageFilters) (<-chan *mioty.ULDataMessage, error) {
	ch := make(chan *mioty.ULDataMessage)
	close(ch)
	return ch, nil
}

func TestStreams_OpenWithTheResponseHeaderBeforeAnyMessage(t *testing.T) {
	t.Run("messages", func(t *testing.T) {
		svc := newRetainedService(t, &retainedFakes{})
		stream, calls := newRecordingStream[pb.Message](ownerCtx())

		require.NoError(t, svc.StreamMessages(&pb.StreamMessagesRequest{}, stream))

		assert.Equal(t, []string{streamedHeader, streamedMessage}, *calls)
	})
	t.Run("events with no event yet", func(t *testing.T) {
		svc := newRetainedService(t, &retainedFakes{})
		svc.eventSvc = &recordingStreamSvc{}
		stream, calls := newRecordingStream[pb.Event](ownerCtx())

		require.NoError(t, svc.StreamEvents(&pb.StreamEventsRequest{}, stream))

		assert.Equal(t, []string{streamedHeader}, *calls)
	})
	t.Run("base station messages with no uplink yet", func(t *testing.T) {
		f := &retainedFakes{}
		svc := newRetainedService(t, f)
		svc.msgListingSvc = quietStationListing{retainedMessageListing{f: f}}
		stream, calls := newRecordingStream[pb.BaseStationMessage](ownerCtx())

		require.NoError(t, svc.StreamBaseStationMessages(&pb.StreamBaseStationMessagesRequest{BsEui: retainedBsEUIHex}, stream))

		assert.Equal(t, []string{streamedHeader}, *calls)
	})
}
