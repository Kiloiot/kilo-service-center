package blueprint

import (
	"context"
	"encoding/json"
	"testing"

	pkgblueprint "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

func TestDecode_SpecCheckFailuresKeepTheirTokens(t *testing.T) {
	undefinedComponent := `{"version":"1.0","typeEui":"70B3D59CD0000095",
		"component":{"known":{"size":8,"type":"uint"}},
		"uplink":[{"id":0,"payload":[{"name":"t","component":"missing"}]}]}`
	tests := []struct {
		name       string
		spec       string
		wantToken  string
		wantDetail string
	}{
		{
			name:      "no uplink and no downlink",
			spec:      `{"version":"1.0","typeEui":"70B3D59CD0000095"}`,
			wantToken: pkgblueprint.ErrMissingUplinkDefs,
		},
		{
			name: "duplicate format id",
			spec: `{"version":"1.0","typeEui":"70B3D59CD0000095",
				"uplink":[{"id":1,"payload":[{"name":"a","size":8,"type":"uint"}]},
				          {"id":1,"payload":[{"name":"b","size":8,"type":"uint"}]}]}`,
			wantToken:  pkgblueprint.ErrDuplicateFormatID,
			wantDetail: "duplicate uplink format ID: 1",
		},
		{
			name:      "undefined component",
			spec:      undefinedComponent,
			wantToken: pkgblueprint.ErrInvalidBlueprintJSON,
			wantDetail: pkgblueprint.ResolveErrorMessage(pkgblueprint.ErrComponentRefNotFound) +
				`: component "t" references undefined key "missing"`,
		},
	}
	svc := NewDecoderService(logger.NewNop())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := svc.Decode(context.Background(),
				&models.Blueprint{SpecJSON: json.RawMessage(tt.spec)}, []byte{0x01}, 1, nil)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if res.Success || res.ErrorCode != tt.wantToken || res.ErrorDetail != tt.wantDetail {
				t.Fatalf("result = %+v, want token %q detail %q", res, tt.wantToken, tt.wantDetail)
			}
		})
	}
}
