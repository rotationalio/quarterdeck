package backend

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/tidal"
)

// Ensure that tidalErr correctly maps errors to Quarterdeck errors, preserving
// the original error cause when possible.
func TestTidalErr(t *testing.T) {
	unknown := fmt.Errorf("mock: unknown error")

	tests := []struct {
		name     string
		in       error
		want     error
		wantSame bool
	}{
		// preserves nil
		{
			name: "nil",
			in:   nil,
			want: nil,
		},
		// preserves quarterdeck domain errors
		{
			name:     "passthrough zero valued not null",
			in:       qerrors.ErrZeroValuedNotNull,
			want:     qerrors.ErrZeroValuedNotNull,
			wantSame: true,
		},
		{
			name:     "passthrough no id on create",
			in:       qerrors.ErrNoIDOnCreate,
			want:     qerrors.ErrNoIDOnCreate,
			wantSame: true,
		},
		{
			name:     "passthrough missing id",
			in:       qerrors.ErrMissingID,
			want:     qerrors.ErrMissingID,
			wantSame: true,
		},
		{
			name:     "passthrough missing reference",
			in:       qerrors.ErrMissingReference,
			want:     qerrors.ErrMissingReference,
			wantSame: true,
		},
		{
			name:     "passthrough type mismatch",
			in:       qerrors.ErrTypeMismatch,
			want:     qerrors.ErrTypeMismatch,
			wantSame: true,
		},
		{
			name:     "passthrough too soon",
			in:       qerrors.ErrTooSoon,
			want:     qerrors.ErrTooSoon,
			wantSame: true,
		},
		{
			name:     "passthrough not authorized",
			in:       qerrors.ErrNotAuthorized,
			want:     qerrors.ErrNotAuthorized,
			wantSame: true,
		},
		{
			name:     "passthrough not found",
			in:       qerrors.ErrNotFound,
			want:     qerrors.ErrNotFound,
			wantSame: true,
		},
		// maps/wraps tidal/sql errors to quarterdeck errors
		{
			name: "map tidal missing id",
			in:   tidal.ErrMissingID,
			want: qerrors.ErrMissingID,
		},
		{
			name: "map tidal not found",
			in:   tidal.ErrNotFound,
			want: qerrors.ErrNotFound,
		},
		{
			name: "map sql no rows",
			in:   sql.ErrNoRows,
			want: qerrors.ErrNotFound,
		},
		{
			name: "wrap unknown error",
			in:   unknown,
			want: qerrors.ErrDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tidalErr(tt.in)
			if tt.want == nil {
				require.NoError(t, got)
				return
			}
			require.ErrorIs(t, got, tt.want)
			if tt.wantSame {
				require.ErrorIs(t, got, tt.in)
			}
		})
	}

	t.Run("wrap unknown error preserves cause", func(t *testing.T) {
		unknown := fmt.Errorf("driver: connection reset")
		got := tidalErr(unknown)
		require.ErrorIs(t, got, qerrors.ErrDatabase)
		require.ErrorIs(t, got, unknown)
	})
}
