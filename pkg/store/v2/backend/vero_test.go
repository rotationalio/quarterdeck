package backend_test

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/suitetest"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/vero"
)

//=============================================================================
// Vero Token Store Tests
//=============================================================================

// TestVeroWorkflow verifies end-to-end create, sign, update, verify, and delete token flow.
func (s *storeSuite) TestVeroWorkflow() {
	require := s.Require()

	// Setup: unsent verify-email token for a fixture user.
	record := &models.VeroToken{
		TokenType:  enum.TokenTypeVerifyEmail,
		ResourceID: ulid.NullULID{ULID: ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"), Valid: true},
		Email:      "gary@example.com",
		Expiration: time.Now().Add(24 * time.Hour),
		Signature:  nil,
		SentOn:     sql.NullTime{Valid: false},
	}

	// Action: create token, sign it, and persist signature/sent timestamp.
	created, err := s.store.CreateVeroToken(s.Context(), record)
	require.NoError(err)
	record = created

	token, err := vero.New(record.ID[:], record.Expiration)
	require.NoError(err)

	var verify vero.VerificationToken
	verify, record.Signature, err = token.Sign()
	require.NoError(err)
	record.SentOn = sql.NullTime{Valid: true, Time: time.Now().In(time.UTC)}

	err = s.store.UpdateVeroToken(s.Context(), record)
	require.NoError(err)

	// Assert: stored signature verifies and token can be deleted.
	retrieved, err := s.store.RetrieveVeroToken(s.Context(), record.ID)
	require.NoError(err)

	secure, err := retrieved.Signature.Verify(verify)
	require.NoError(err)
	require.True(secure)

	err = s.store.DeleteVeroToken(s.Context(), record.ID)
	require.NoError(err)
}

// TestCreateVeroToken verifies validation and successful token creation.
func (s *storeSuite) TestCreateVeroToken() {
	require := s.Require()

	s.Run("NoIDOnCreate", func() {
		token := &models.VeroToken{}
		token.ID = ulid.Make()

		_, err := s.store.CreateVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrNoIDOnCreate)
	})

	s.Run("HappyPath", func() {
		tokenCount := s.count("vero_tokens")

		token := &models.VeroToken{
			TokenType:  enum.TokenTypeVerifyEmail,
			ResourceID: ulid.NullULID{ULID: ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"), Valid: true},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
			Signature:  nil,
			SentOn:     sql.NullTime{Valid: false},
		}

		created, err := s.store.CreateVeroToken(s.Context(), token)
		require.NoError(err)
		require.NotEqual(ulid.Zero, created.ID)
		require.NotZero(created.Created)
		require.NotZero(created.Modified)
		require.Equal(tokenCount+1, s.count("vero_tokens"))
	})
}

// TestRetrieveVeroToken verifies lookup by ID and not-found cases.
func (s *storeSuite) TestRetrieveVeroToken() {
	require := s.Require()

	s.Run("NoID", func() {
		token, err := s.store.RetrieveVeroToken(s.Context(), ulid.Zero)
		require.ErrorIs(err, errors.ErrNotFound)
		require.Nil(token)
	})

	s.Run("HappyPath", func() {
		tokenID := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		token, err := s.store.RetrieveVeroToken(s.Context(), tokenID)
		require.NoError(err)

		require.Equal(token.ID, tokenID)
		require.Equal(token.TokenType, enum.TokenTypeResetPassword)
		require.Equal(token.ResourceID, ulid.NullULID{ULID: ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T"), Valid: true})
		require.Equal(token.Email, "observer@example.com")
		require.Equal(time.Date(2024, time.November, 16, 17, 43, 53, 0, time.UTC), token.Expiration)
		require.Len(token.Signature.Signature(), 32)
		require.Equal(time.Date(2024, time.November, 16, 17, 28, 45, 0, time.UTC), token.SentOn.Time)
		require.Equal(time.Date(2024, time.November, 16, 17, 28, 57, 0, time.UTC), token.Created)
		require.Equal(time.Date(2024, time.November, 16, 17, 28, 57, 0, time.UTC), token.Modified)
	})

	s.Run("NotFound", func() {
		token, err := s.store.RetrieveVeroToken(s.Context(), ulid.Make())
		require.ErrorIs(err, errors.ErrNotFound)
		require.Nil(token)
	})
}

// TestUpdateVeroToken verifies field updates preserve created timestamp and reject invalid IDs.
func (s *storeSuite) TestUpdateVeroToken() {
	require := s.Require()

	token, err := s.store.RetrieveVeroToken(s.Context(), ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D"))
	require.NoError(err)
	staleCreated := token.Created
	token.TokenType = enum.TokenTypeTeamInvite
	token.ResourceID = ulid.NullULID{Valid: true, ULID: ulid.Make()}
	token.Email = "different@zoom.com"
	token.Expiration = time.Now().Add(24 * time.Hour).In(time.UTC)
	token.Signature = nil
	token.SentOn = sql.NullTime{Valid: true, Time: time.Now().Add(-1231 * time.Millisecond).In(time.UTC)}

	s.Run("NoID", func() {
		err := s.store.UpdateVeroToken(s.Context(), &models.VeroToken{})
		require.ErrorIs(err, errors.ErrMissingID)
	})

	s.Run("HappyPath", func() {
		tokenCount := s.count("vero_tokens")

		err := s.store.UpdateVeroToken(s.Context(), token)
		require.NoError(err)
		require.Equal(tokenCount, s.count("vero_tokens"))

		cmpt, err := s.store.RetrieveVeroToken(s.Context(), token.ID)
		require.NoError(err)

		require.Equal(token.ID, cmpt.ID)
		require.Equal(token.TokenType, cmpt.TokenType)
		require.Equal(token.ResourceID, cmpt.ResourceID)
		require.Equal(token.Email, cmpt.Email)
		suitetest.EqualTime(s.T(), token.Expiration, cmpt.Expiration)
		require.Equal(token.Signature, cmpt.Signature)
		suitetest.EqualTime(s.T(), token.SentOn.Time, cmpt.SentOn.Time)
		require.Equal(staleCreated, cmpt.Created)
		require.WithinDuration(time.Now(), cmpt.Modified, 5*time.Second)
	})

	s.Run("NotFound", func() {
		token.ID = ulid.Make()
		err := s.store.UpdateVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrNotFound)
	})
}

// TestDeleteVeroToken verifies deletion and error handling for missing/unknown IDs.
func (s *storeSuite) TestDeleteVeroToken() {
	require := s.Require()

	s.Run("NoID", func() {
		err := s.store.DeleteVeroToken(s.Context(), ulid.Zero)
		require.ErrorIs(err, errors.ErrMissingID)
	})

	s.Run("HappyPath", func() {
		tokenCount := s.count("vero_tokens")
		tokenID := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		err := s.store.DeleteVeroToken(s.Context(), tokenID)
		require.NoError(err)
		require.Equal(tokenCount-1, s.count("vero_tokens"))
	})

	s.Run("NotFound", func() {
		tokenID := ulid.Make()
		err := s.store.DeleteVeroToken(s.Context(), tokenID)
		require.ErrorIs(err, errors.ErrNotFound)
	})
}

// TestCreateResetPasswordVeroToken verifies reset-password-specific validation and rate limiting.
func (s *storeSuite) TestCreateResetPasswordVeroToken() {
	require := s.Require()

	s.Run("NoIDOnCreate", func() {
		token := &models.VeroToken{}
		token.ID = ulid.Make()

		_, err := s.store.CreateResetPasswordVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrNoIDOnCreate)
	})

	s.Run("TypeMismatch", func() {
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeVerifyEmail,
			ResourceID: ulid.NullULID{ULID: ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"), Valid: true},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
			Signature:  nil,
			SentOn:     sql.NullTime{Valid: false},
		}

		_, err := s.store.CreateResetPasswordVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrTypeMismatch)
	})

	s.Run("ErrMissingReference_InvalidResourceID", func() {
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeResetPassword,
			ResourceID: ulid.NullULID{Valid: false},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
			Signature:  nil,
			SentOn:     sql.NullTime{Valid: false},
		}

		_, err := s.store.CreateResetPasswordVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrMissingReference)
	})

	s.Run("ErrMissingReference_ZeroResourceID", func() {
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeResetPassword,
			ResourceID: ulid.NullULID{Valid: true},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
			Signature:  nil,
			SentOn:     sql.NullTime{Valid: false},
		}

		_, err := s.store.CreateResetPasswordVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrMissingReference)
	})

	s.Run("TooSoon", func() {
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeResetPassword,
			ResourceID: ulid.NullULID{Valid: true, ULID: ulid.MakeSecure()},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
			Signature:  nil,
			SentOn:     sql.NullTime{Valid: false},
		}

		// Action: create first reset token, then attempt immediate duplicate.
		created, err := s.store.CreateResetPasswordVeroToken(s.Context(), token)
		require.NoError(err)

		token = &models.VeroToken{
			TokenType:  enum.TokenTypeResetPassword,
			ResourceID: created.ResourceID,
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
			Signature:  nil,
			SentOn:     sql.NullTime{Valid: false},
		}
		_, err = s.store.CreateResetPasswordVeroToken(s.Context(), token)
		// Assert: rate limit rejects second token for same resource.
		require.ErrorIs(err, errors.ErrTooSoon)
	})

	s.Run("HappyPath", func() {
		tokenCount := s.count("vero_tokens")

		// Setup: valid reset-password token for a fixture user.
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeResetPassword,
			ResourceID: ulid.NullULID{ULID: ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"), Valid: true},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
			Signature:  nil,
			SentOn:     sql.NullTime{Valid: false},
		}

		// Action: create reset-password token.
		created, err := s.store.CreateResetPasswordVeroToken(s.Context(), token)
		require.NoError(err)
		// Assert: token persisted with generated ID and timestamps.
		require.NotEqual(ulid.Zero, created.ID)
		require.NotZero(created.Created)
		require.NotZero(created.Modified)
		require.Equal(tokenCount+1, s.count("vero_tokens"))
	})
}

// TestCompletePasswordReset verifies password reset completion, token cleanup, and error cases.
func (s *storeSuite) TestCompletePasswordReset() {
	require := s.Require()
	userID := ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A")

	s.Run("NotFound", func() {
		// Action: complete reset with unknown token ID.
		err := s.store.CompletePasswordReset(s.Context(), ulid.Make(), "new-password")
		// Assert: token not found.
		require.ErrorIs(err, errors.ErrNotFound)
	})

	s.Run("TypeMismatch", func() {
		// Setup: verify-email token (wrong type for password reset).
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeVerifyEmail,
			ResourceID: ulid.NullULID{ULID: userID, Valid: true},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
		}
		created, err := s.store.CreateVeroToken(s.Context(), token)
		require.NoError(err)

		// Action: attempt password reset with wrong token type.
		err = s.store.CompletePasswordReset(s.Context(), created.ID, "new-password")
		// Assert: type mismatch rejected.
		require.ErrorIs(err, errors.ErrTypeMismatch)
	})

	s.Run("Expired", func() {
		// Setup: expired reset-password token.
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeResetPassword,
			ResourceID: ulid.NullULID{ULID: userID, Valid: true},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(-time.Hour),
		}
		created, err := s.store.CreateVeroToken(s.Context(), token)
		require.NoError(err)

		// Action: attempt password reset with expired token.
		err = s.store.CompletePasswordReset(s.Context(), created.ID, "new-password")
		// Assert: expired token rejected.
		require.ErrorIs(err, errors.ErrExpiredToken)
	})

	s.Run("HappyPath", func() {
		newPassword := "$argon2id$v=19$m=65536,t=1,p=2$DT/LSMZjHhVlprmPaBSCcg==$UKT1g5gqWvKhiBC8gywVU6zepCEew0x3IW9vTWnlVlg="

		// Setup: valid reset-password token for fixture user.
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeResetPassword,
			ResourceID: ulid.NullULID{ULID: userID, Valid: true},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
		}
		created, err := s.store.CreateResetPasswordVeroToken(s.Context(), token)
		require.NoError(err)

		// Action: complete password reset.
		err = s.store.CompletePasswordReset(s.Context(), created.ID, newPassword)
		require.NoError(err)

		// Assert: user password updated and token removed.
		user, err := s.store.RetrieveUser(s.Context(), userID)
		require.NoError(err)
		require.Equal(newPassword, user.Password)

		_, err = s.store.RetrieveVeroToken(s.Context(), created.ID)
		require.ErrorIs(err, errors.ErrNotFound)
	})
}

// TestRetrieveTeamInviteVeroToken verifies lookup of team invite token by resource ID.
func (s *storeSuite) TestRetrieveTeamInviteVeroToken() {
	require := s.Require()

	// Setup + action: create a team-invite token for a new user resource.
	userID := ulid.MakeSecure()
	token := &models.VeroToken{
		TokenType:  enum.TokenTypeTeamInvite,
		ResourceID: ulid.NullULID{Valid: true, ULID: userID},
		Email:      "invite@example.com",
		Expiration: time.Now().Add(48 * time.Hour),
	}

	created, err := s.store.CreateTeamInviteVeroToken(s.Context(), token)
	require.NoError(err)

	// Assert: token is retrievable by resource ID and type.
	got, err := s.store.RetrieveVeroTokenByResource(s.Context(), userID, enum.TokenTypeTeamInvite)
	require.NoError(err)
	require.Equal(created.ID, got.ID)
	require.Equal(enum.TokenTypeTeamInvite, got.TokenType)

	_, err = s.store.RetrieveVeroTokenByResource(s.Context(), ulid.Make(), enum.TokenTypeTeamInvite)
	require.ErrorIs(err, errors.ErrNotFound)
}
