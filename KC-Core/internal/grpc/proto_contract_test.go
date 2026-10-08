package grpc

import (
	"testing"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Every 64-bit identity or nanosecond field must reach browser clients as a
// string: JavaScript numbers lose precision above 2^53.
func TestProtoContract_SixtyFourBitFieldsCarryJSString(t *testing.T) {
	cases := []struct {
		message string
		field   string
		number  protoreflect.FieldNumber
		kind    protoreflect.Kind
	}{
		{"BaseStationReceptionInfo", "rx_time", 2, protoreflect.Int64Kind},
		{"Message", "op_id", 24, protoreflect.Int64Kind},
		{"UpdatePendingDownlinkRequest", "que_id", 2, protoreflect.Int64Kind},
		{"ListDownlinkQueueRequest", "que_id", 7, protoreflect.Int64Kind},
		{"DownlinkMessage", "tx_time", 12, protoreflect.Int64Kind},
		{"DownlinkMessage", "que_id", 22, protoreflect.Int64Kind},
		{"GetDownlinkResultsRequest", "que_id", 9, protoreflect.Int64Kind},
		{"BaseStationStatusResponse", "op_id", 3, protoreflect.Int64Kind},
		{"InitiatePingResponse", "op_id", 3, protoreflect.Int64Kind},
		{"ReleaseInfo", "sc_eui", 9, protoreflect.Uint64Kind},
		{"DLRXStatus", "rx_time", 3, protoreflect.Int64Kind},
		{"DLRXStatusQuery", "op_id", 3, protoreflect.Int64Kind},
		{"ListEventsRequest", "op_id", 8, protoreflect.Int64Kind},
		{"ScaciSession", "last_op_id_ac", 11, protoreflect.Int64Kind},
		{"ScaciSession", "last_op_id_sc", 12, protoreflect.Int64Kind},
		{"ScaciQueueEntry", "que_id", 8, protoreflect.Int64Kind},
	}
	for _, tc := range cases {
		t.Run(tc.message+"."+tc.field, func(t *testing.T) {
			msg := pb.File_core_proto.Messages().ByName(protoreflect.Name(tc.message))
			require.NotNil(t, msg, "message %s missing", tc.message)
			fd := msg.Fields().ByName(protoreflect.Name(tc.field))
			require.NotNil(t, fd, "field %s.%s missing", tc.message, tc.field)
			require.Equal(t, tc.number, fd.Number())
			require.Equal(t, tc.kind, fd.Kind())
			opts, ok := fd.Options().(*descriptorpb.FieldOptions)
			require.True(t, ok)
			require.Equal(t, descriptorpb.FieldOptions_JS_STRING, opts.GetJstype())
		})
	}
}

// The legacy implicit-presence filter stays on the wire for old clients; the
// tri-state filter lives on its own explicit-presence field.
func TestProtoContract_ScaciSessionResumeFilterFields(t *testing.T) {
	msg := pb.File_core_proto.Messages().ByName("ListScaciSessionsRequest")
	require.NotNil(t, msg)

	legacy := msg.Fields().ByName("can_resume")
	require.NotNil(t, legacy)
	require.Equal(t, protoreflect.FieldNumber(4), legacy.Number())
	require.False(t, legacy.HasPresence(), "legacy field keeps implicit presence")
	legacyOpts, ok := legacy.Options().(*descriptorpb.FieldOptions)
	require.True(t, ok)
	require.True(t, legacyOpts.GetDeprecated())

	filter := msg.Fields().ByName("can_resume_filter")
	require.NotNil(t, filter)
	require.Equal(t, protoreflect.FieldNumber(5), filter.Number())
	require.True(t, filter.HasPresence(), "filter field must distinguish absent from false")
}
