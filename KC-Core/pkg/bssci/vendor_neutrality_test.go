package bssci

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// TestConnect_VendorAndModelDoNotSteerNegotiation pins that the con payload's
// vendor and model strings are opaque metadata: the negotiated version and
// the response shape are identical for every value, so no base station
// manufacturer receives special handling.
func TestConnect_VendorAndModelDoNotSteerNegotiation(t *testing.T) {
	samples := []struct{ vendor, model string }{
		{"", ""},
		{"vendor-a", "model-1"},
		{"Vendor B GmbH", "BS-2000"},
		{"some unknown maker", "prototype"},
	}
	var reference map[string]interface{}
	for _, sample := range samples {
		h := startInteropServer(t, EncodingMessagePack)
		payload := connectPayload("1.1.0", uint64(0xCAFECAFECAFECAFE))
		payload["vendor"] = sample.vendor
		payload["model"] = sample.model
		h.writeFrame(payload)

		conRsp := h.readFrame()
		require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp), "vendor %q model %q", sample.vendor, sample.model)
		if reference == nil {
			reference = conRsp
			continue
		}
		require.Equal(t, reference["version"], conRsp["version"], "negotiated version must not depend on vendor %q model %q", sample.vendor, sample.model)
		require.Equal(t, len(reference), len(conRsp), "response shape must not depend on vendor %q model %q", sample.vendor, sample.model)
	}
}
