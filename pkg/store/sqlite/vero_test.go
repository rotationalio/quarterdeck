package sqlite_test

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/vero"
)

func (s *storeTestSuite) TestVeroWorkflow() {
	if s.ReadOnly() {
		s.T().Skip("skipping Vero workflow test in read-only mode")
	}

	// The Vero workflow requires the creation of a VeroToken to get an ID, then
	// updating the token with the signature and sent on date, retrieving it for
	// verification, and finally deleting it. Although these methods are tested
	// independently, this test ensures the entire workflow works as expected.
	require := s.Require()

	record := &models.VeroToken{
		TokenType:  enum.TokenTypeVerifyEmail,
		ResourceID: ulid.NullULID{ULID: ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"), Valid: true},
		Email:      "gary@example.com",
		Expiration: time.Now().Add(24 * time.Hour),
		Signature:  nil,                        // Signature will be set after creation
		SentOn:     sql.NullTime{Valid: false}, // SentOn will be set after creation
	}

	// Create the database record to get an ID
	err := s.db.CreateVeroToken(s.Context(), record)
	require.NoError(err, "should successfully create a VeroToken")

	// Now create the token and "send" the email, updating the record.
	token, err := vero.New(record.ID[:], record.Expiration)
	require.NoError(err, "should successfully create a new Vero token")

	var verify vero.VerificationToken
	verify, record.Signature, err = token.Sign()
	require.NoError(err, "should successfully sign the Vero token")
	record.SentOn = sql.NullTime{Valid: true, Time: time.Now().In(time.UTC)}

	err = s.db.UpdateVeroToken(s.Context(), record)
	require.NoError(err, "should successfully update the VeroToken with signature and sentOn")

	// Retrieve the token to verify it it with the verify token
	retrieved, err := s.db.RetrieveVeroToken(s.Context(), record.ID)
	require.NoError(err, "should successfully retrieve the VeroToken")

	secure, err := retrieved.Signature.Verify(verify)
	require.NoError(err, "should successfully verify the VeroToken signature")
	require.True(secure, "should verify the VeroToken signature")

	// Now delete the token
	err = s.db.DeleteVeroToken(s.Context(), record.ID)
	require.NoError(err, "should successfully delete the VeroToken")
}

func (s *storeTestSuite) TestCreateVeroToken() {
	require := s.Require()

	s.Run("NoIDOnCreate", func() {
		token := &models.VeroToken{
			Model: models.Model{
				ID:       ulid.Make(),
				Created:  time.Now(),
				Modified: time.Now(),
			},
		}

		err := s.db.CreateVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrNoIDOnCreate, "should not allow creating a VeroToken with an ID")
	})

	s.Run("ReadOnly", func() {
		if !s.ReadOnly() {
			s.T().Skip("skipping read-only create test in read-write mode")
		}

		// Create a token to prepare a signature.
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeVerifyEmail,
			ResourceID: ulid.NullULID{ULID: ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"), Valid: true},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
			Signature:  nil,
			SentOn:     sql.NullTime{Valid: false},
		}

		err := s.db.CreateVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrReadOnly, "should return read-only error when trying to create a VeroToken in read-only mode")
	})

	s.Run("HappyPath", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		tokenCount := s.Count("vero_tokens")

		// Create a token to prepare a signature.
		token := &models.VeroToken{
			TokenType:  enum.TokenTypeVerifyEmail,
			ResourceID: ulid.NullULID{ULID: ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"), Valid: true},
			Email:      "gary@example.com",
			Expiration: time.Now().Add(24 * time.Hour),
			Signature:  nil,
			SentOn:     sql.NullTime{Valid: false},
		}

		err := s.db.CreateVeroToken(s.Context(), token)
		require.NoError(err, "should successfully create a VeroToken")
		require.NotEqual(ulid.Zero, token.ID, "should assign a new ID to the VeroToken")
		require.NotZero(token.Created, "should set Created timestamp")
		require.NotZero(token.Modified, "should set Modified timestamp")

		require.Equal(tokenCount+1, s.Count("vero_tokens"), "should create a new VeroToken")
	})
}

func (s *storeTestSuite) TestRetrieveVeroToken() {
	require := s.Require()

	s.Run("NoID", func() {
		token, err := s.db.RetrieveVeroToken(s.Context(), ulid.Zero)
		require.ErrorIs(err, errors.ErrMissingID, "should return error when retrieving VeroToken with zero ID")
		require.Nil(token, "should not return a token for zero ID")
	})

	s.Run("HappyPath", func() {
		tokenID := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		token, err := s.db.RetrieveVeroToken(s.Context(), tokenID)
		require.NoError(err, "should successfully retrieve existing VeroToken")

		require.Equal(token.ID, tokenID, "should return the correct VeroToken ID")
		require.Equal(token.TokenType, enum.TokenTypeResetPassword, "should return the correct TokenType")
		require.Equal(token.ResourceID, ulid.NullULID{ULID: ulid.MustParse("01HWQE3N4S6PZGKNCH7E617N8T"), Valid: true}, "should return the correct ResourceID")
		require.Equal(token.Email, "observer@example.com", "should return the correct Email")
		require.Equal(time.Date(2024, time.November, 16, 17, 43, 53, 0, time.UTC), token.Expiration, "should return the correct Expiration time")
		require.Len(token.Signature.Signature(), 32, "should return the correct Signature")
		require.Equal(time.Date(2024, time.November, 16, 17, 28, 45, 0, time.UTC), token.SentOn.Time, "should return the correct SentOn time")
		require.Equal(time.Date(2024, time.November, 16, 17, 28, 57, 0, time.UTC), token.Created, "should return the correct Created time")
		require.Equal(time.Date(2024, time.November, 16, 17, 28, 57, 0, time.UTC), token.Modified, "should return the correct Modified time")
	})

	s.Run("NotFound", func() {
		token, err := s.db.RetrieveVeroToken(s.Context(), ulid.Make())
		require.ErrorIs(err, errors.ErrNotFound, "should return not found error when retrieving non-existent VeroToken")
		require.Nil(token, "should not return a token for non-existent VeroToken")
	})
}

func (s *storeTestSuite) TestUpdateVeroToken() {
	require := s.Require()

	s.Run("NoID", func() {
		err := s.db.UpdateVeroToken(s.Context(), &models.VeroToken{})
		require.ErrorIs(err, errors.ErrMissingID, "should return error when updating VeroToken with zero ID")
	})

	token := &models.VeroToken{
		Model: models.Model{
			ID:       ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D"),           // Existing token ID for testing
			Created:  time.Date(1998, time.April, 8, 7, 42, 12, 0, time.UTC), // Created time should not be modified
			Modified: time.Date(1998, time.April, 8, 7, 42, 12, 0, time.UTC), // Modified time should be set to now
		},
		TokenType:  enum.TokenTypeTeamInvite,
		ResourceID: ulid.NullULID{Valid: true, ULID: ulid.Make()},
		Email:      "different@zoom.com",
		Expiration: time.Now().Add(24 * time.Hour).In(time.UTC),
		Signature:  nil,
		SentOn:     sql.NullTime{Valid: true, Time: time.Now().Add(-1231 * time.Millisecond).In(time.UTC)},
	}

	s.Run("ReadOnly", func() {
		if !s.ReadOnly() {
			s.T().Skip("skipping read-only update test in read-write mode")
		}

		err := s.db.UpdateVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrReadOnly, "should return read-only error when trying to update a VeroToken in read-only mode")
	})

	s.Run("HappyPath", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping update test in read-only mode")
		}

		tokenCount := s.Count("vero_tokens")

		err := s.db.UpdateVeroToken(s.Context(), token)
		require.NoError(err, "should successfully update existing VeroToken")
		require.Equal(tokenCount, s.Count("vero_tokens"), "should not change the count of vero_tokens")

		cmpt, err := s.db.RetrieveVeroToken(s.Context(), token.ID)
		require.NoError(err, "should successfully retrieve updated VeroToken")

		require.Equal(token.ID, cmpt.ID, "should return the correct VeroToken ID after update")
		require.Equal(token.TokenType, cmpt.TokenType, "should return the correct TokenType after update")
		require.Equal(token.ResourceID, cmpt.ResourceID, "should return the correct ResourceID after update")
		require.Equal(token.Email, cmpt.Email, "should return the correct Email after update")
		require.Equal(token.Expiration, cmpt.Expiration, "should return the correct Expiration time after update")
		require.Equal(token.Signature, cmpt.Signature, "should update Signature after update")
		require.Equal(token.SentOn.Time, cmpt.SentOn.Time, "should return the correct SentOn time after update")
		require.NotEqual(token.Created, cmpt.Created, "should not change Created time after update")
		require.WithinDuration(time.Now(), token.Modified, 5*time.Second, "should set Modified time to now after update")
	})

	s.Run("NotFound", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping update test in read-only mode")
		}

		token.ID = ulid.Make() // Use a non-existent ID
		err := s.db.UpdateVeroToken(s.Context(), token)
		require.ErrorIs(err, errors.ErrNotFound, "should return not found error when updating non-existent VeroToken")
	})
}

func (s *storeTestSuite) TestDeleteVeroToken() {
	require := s.Require()

	s.Run("NoID", func() {
		err := s.db.DeleteVeroToken(s.Context(), ulid.Zero)
		require.ErrorIs(err, errors.ErrMissingID, "should return error when deleting VeroToken with zero ID")
	})

	s.Run("ReadOnly", func() {
		if !s.ReadOnly() {
			s.T().Skip("skipping read-only delete test in read-write mode")
		}

		tokenID := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		err := s.db.DeleteVeroToken(s.Context(), tokenID)
		require.ErrorIs(err, errors.ErrReadOnly, "should return read-only error when trying to delete a VeroToken in read-only mode")
	})

	s.Run("HappyPath", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping delete test in read-only mode")
		}

		tokenCount := s.Count("vero_tokens")
		tokenID := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		err := s.db.DeleteVeroToken(s.Context(), tokenID)
		require.NoError(err, "should successfully delete existing VeroToken")
		require.Equal(tokenCount-1, s.Count("vero_tokens"), "should decrease the count of vero_tokens by 1")
	})

	s.Run("NotFound", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping delete test in read-only mode")
		}
		tokenID := ulid.Make()
		err := s.db.DeleteVeroToken(s.Context(), tokenID)
		require.ErrorIs(err, errors.ErrNotFound, "should return not found error when deleting non-existent VeroToken")
	})
}
