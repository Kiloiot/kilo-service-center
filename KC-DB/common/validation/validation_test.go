package validation

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEUI(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  uint64
		err   error
	}{
		{name: "plain hex", input: "70B3D59CD00009E6", want: 0x70B3D59CD00009E6},
		{name: "lowercase", input: "70b3d59cd00009e6", want: 0x70B3D59CD00009E6},
		{name: "dashed", input: "70-B3-D5-9C-D0-00-09-E6", want: 0x70B3D59CD00009E6},
		{name: "colons", input: "70:B3:D5:9C:D0:00:09:E6", want: 0x70B3D59CD00009E6},
		{name: "empty", input: "", err: errors.ErrMissingField},
		{name: "too short", input: "010203", err: errors.ErrInvalidEUI},
		{name: "too long", input: "70B3D59CD00009E6FF", err: errors.ErrInvalidEUI},
		{name: "hex prefix rejected", input: "0x70B3D59CD00009E6", err: errors.ErrInvalidEUI},
		{name: "non hex", input: "70B3D59CD00009ZZ", err: errors.ErrInvalidEUI},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseEUI(tc.input)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseEUIBytes(t *testing.T) {
	t.Parallel()
	got, err := ParseEUIBytes("70B3D59CD00009E6")
	require.NoError(t, err)
	assert.Equal(t, []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE6}, got)

	_, err = ParseEUIBytes("010203")
	require.ErrorIs(t, err, errors.ErrInvalidEUI)
}
