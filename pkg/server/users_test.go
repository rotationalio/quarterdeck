package server

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/store/v1/models"
)

// TestWelcomeEmailRateLimited checks the resend cooldown uses SentOn.
func TestWelcomeEmailRateLimited(t *testing.T) {
	t.Run("NoSentOn", func(t *testing.T) {
		require.False(t, welcomeEmailRateLimited(&models.VeroToken{}))
	})

	t.Run("RecentSend", func(t *testing.T) {
		record := &models.VeroToken{
			SentOn: sql.NullTime{Valid: true, Time: time.Now().Add(-5 * time.Minute)},
		}
		require.True(t, welcomeEmailRateLimited(record))
	})

	t.Run("AfterCooldown", func(t *testing.T) {
		record := &models.VeroToken{
			SentOn: sql.NullTime{Valid: true, Time: time.Now().Add(-20 * time.Minute)},
		}
		require.False(t, welcomeEmailRateLimited(record))
	})
}
