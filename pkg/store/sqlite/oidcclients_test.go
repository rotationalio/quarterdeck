package sqlite_test

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

const (
	fullMetadataClientID = "OidcClient1FullMetadata"
	minimalClientID      = "OidcClient2Minimal"
	keyholderUserULID    = "01JPYRNYMEHNEZCS0JYX1CP57A" // Keyholder from testdata, for Create tests
)

func (s *storeTestSuite) TestListOIDCClients() {
	s.Run("Maximal", func() {
		require := s.Require()
		out, err := s.db.ListOIDCClients(s.Context(), nil)
		require.NoError(err, "should be able to list OIDC clients")
		require.NotNil(out, "should return an OIDC client list")
		require.Len(out.OIDCClients, 2, "list should return 2 non-revoked clients")

		var full *models.OIDCClient
		for _, c := range out.OIDCClients {
			if c.ClientID == fullMetadataClientID {
				full = c
				break
			}
		}
		require.NotNil(full, "full-metadata client should be in list")
		require.Equal("Full Metadata OIDC Client", full.ClientName)
		require.True(full.ClientURI.Valid)
		require.Equal("https://example.com", full.ClientURI.String)
		require.True(full.LogoURI.Valid)
		require.Equal("https://example.com/logo.png", full.LogoURI.String)
		require.True(full.PolicyURI.Valid)
		require.Equal("https://example.com/policy", full.PolicyURI.String)
		require.True(full.TOSURI.Valid)
		require.Equal("https://example.com/tos", full.TOSURI.String)
		require.Len(full.RedirectURIs, 2)
		require.Equal([]string{"https://example.com/callback", "https://app.example.com/cb"}, full.RedirectURIs)
		require.Len(full.Contacts, 2)
		require.True(full.Contacts[0].Valid)
		require.Equal("admin@example.com", full.Contacts[0].String)
		require.True(full.Contacts[1].Valid)
		require.Equal("support@example.com", full.Contacts[1].String)
		require.Equal(fullMetadataClientID, full.ClientID)
		require.Empty(full.Secret, "list should not return secret")
		require.False(full.CreatedBy.IsZero())
		require.False(full.Revoked.Valid)
		require.False(full.Created.IsZero())
		require.False(full.Modified.IsZero())
	})

	s.Run("Minimal", func() {
		require := s.Require()
		out, err := s.db.ListOIDCClients(s.Context(), nil)
		require.NoError(err, "should be able to list OIDC clients")

		var minimal *models.OIDCClient
		for _, c := range out.OIDCClients {
			if c.ClientID == minimalClientID {
				minimal = c
				break
			}
		}
		require.NotNil(minimal, "minimal client should be in list")
		require.Empty(minimal.ClientName)
		require.False(minimal.ClientURI.Valid)
		require.False(minimal.LogoURI.Valid)
		require.False(minimal.PolicyURI.Valid)
		require.False(minimal.TOSURI.Valid)
		require.NotNil(minimal.RedirectURIs)
		require.Len(minimal.RedirectURIs, 1)
		require.Equal("https://example.com/cb", minimal.RedirectURIs[0])
		require.Nil(minimal.Contacts)
		require.Equal(minimalClientID, minimal.ClientID)
		require.Empty(minimal.Secret)
	})
}

func (s *storeTestSuite) TestCreateOIDCClient() {
	s.Run("NoIDOnCreate", func() {
		client := &models.OIDCClient{
			Model:      models.Model{ID: ulid.Make()},
			ClientName: "Test",
			ClientID:   "TestClientID123",
			Secret:     "secret",
			CreatedBy:  ulid.MustParse(keyholderUserULID),
		}
		err := s.db.CreateOIDCClient(s.Context(), client)
		s.Require().ErrorIs(err, errors.ErrNoIDOnCreate)
	})

	s.Run("RejectsEmptyClientID", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping in read-only mode")
		}
		// client_id is required; CreateOIDCClient should reject and return validation error.
		client := &models.OIDCClient{
			ClientName:   "Test",
			ClientID:     "",
			Secret:       "secret",
			CreatedBy:    ulid.MustParse(keyholderUserULID),
			RedirectURIs: []string{"https://example.com/cb"},
		}
		err := s.db.CreateOIDCClient(s.Context(), client)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "client_id")
	})

	s.Run("RejectsEmptySecret", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping in read-only mode")
		}
		// secret is required; CreateOIDCClient should reject and return validation error.
		client := &models.OIDCClient{
			ClientName:   "Test",
			ClientID:     "valid-client-id",
			Secret:       "",
			CreatedBy:    ulid.MustParse(keyholderUserULID),
			RedirectURIs: []string{"https://example.com/cb"},
		}
		err := s.db.CreateOIDCClient(s.Context(), client)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "secret")
	})

	s.Run("ReadOnly", func() {
		if !s.ReadOnly() {
			s.T().Skip("skipping create read-only error test in read-write mode")
		}
		client := &models.OIDCClient{
			ClientName:   "Test",
			ClientURI:    sql.NullString{Valid: true, String: "https://example.com"},
			ClientID:     "TestClientID456",
			Secret:       "secret",
			CreatedBy:    ulid.MustParse(keyholderUserULID),
			RedirectURIs: []string{"https://example.com/cb"},
		}
		err := s.db.CreateOIDCClient(s.Context(), client)
		s.Require().ErrorIs(err, errors.ErrReadOnly)
	})

	s.Run("Maximal", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}
		require := s.Require()
		client := &models.OIDCClient{
			ClientName:   "Created Full Client",
			ClientURI:    sql.NullString{Valid: true, String: "https://created.example.com"},
			LogoURI:      sql.NullString{Valid: true, String: "https://created.example.com/logo.png"},
			PolicyURI:    sql.NullString{Valid: true, String: "https://created.example.com/policy"},
			TOSURI:       sql.NullString{Valid: true, String: "https://created.example.com/tos"},
			RedirectURIs: []string{"https://created.example.com/cb", "https://created.example.com/cb2"},
			Contacts:     []sql.NullString{{Valid: true, String: "created@example.com"}, {Valid: true, String: "support@created.example.com"}},
			ClientID:     "CreatedFullOIDCClient",
			Secret:       "$argon2id$v=19$m=65536,t=1,p=2$createdsecretbase64$createdsaltsuffix",
			CreatedBy:    ulid.MustParse(keyholderUserULID),
		}
		err := s.db.CreateOIDCClient(s.Context(), client)
		require.NoError(err)
		require.False(client.ID.IsZero())
		require.WithinDuration(time.Now(), client.Created, 3*time.Second)
		require.WithinDuration(time.Now(), client.Modified, 3*time.Second)

		got, err := s.db.RetrieveOIDCClient(s.Context(), client.ClientID)
		require.NoError(err)
		require.Equal(client.ID, got.ID)
		require.Equal(client.ClientName, got.ClientName)
		require.Equal(client.ClientURI, got.ClientURI)
		require.Equal(client.LogoURI, got.LogoURI)
		require.Equal(client.PolicyURI, got.PolicyURI)
		require.Equal(client.TOSURI, got.TOSURI)
		require.Equal(client.RedirectURIs, got.RedirectURIs)
		require.Len(got.Contacts, 2)
		require.Equal(client.Contacts[0].String, got.Contacts[0].String)
		require.Equal(client.Contacts[1].String, got.Contacts[1].String)
		require.Equal(client.ClientID, got.ClientID)
		require.Equal(client.Secret, got.Secret)
		require.Equal(client.CreatedBy, got.CreatedBy)
		require.WithinDuration(client.Created, got.Created, time.Second)
		require.WithinDuration(client.Modified, got.Modified, time.Second)
	})

	s.Run("Minimal", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}
		require := s.Require()
		client := &models.OIDCClient{
			ClientName:   "",
			ClientURI:    sql.NullString{Valid: false},
			LogoURI:      sql.NullString{Valid: false},
			PolicyURI:    sql.NullString{Valid: false},
			TOSURI:       sql.NullString{Valid: false},
			RedirectURIs: []string{"https://minimal.example.com/cb"},
			Contacts:     nil,
			ClientID:     "CreatedMinimalOIDCClient",
			Secret:       "$argon2id$v=19$m=65536,t=1,p=2$minimalsecret$minimalsalt",
			CreatedBy:    ulid.MustParse(keyholderUserULID),
		}
		err := s.db.CreateOIDCClient(s.Context(), client)
		require.NoError(err)
		require.False(client.ID.IsZero())

		got, err := s.db.RetrieveOIDCClient(s.Context(), client.ClientID)
		require.NoError(err)
		require.Equal(client.ClientName, got.ClientName)
		require.False(got.ClientURI.Valid)
		require.False(got.LogoURI.Valid)
		require.False(got.PolicyURI.Valid)
		require.False(got.TOSURI.Valid)
		require.Equal(client.RedirectURIs, got.RedirectURIs)
		require.Empty(got.Contacts, "contacts should be nil or empty slice")
		require.Equal(client.ClientID, got.ClientID)
		require.Equal(client.Secret, got.Secret)
	})
}

func (s *storeTestSuite) TestRetrieveOIDCClient() {
	s.Run("Success", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), fullMetadataClientID)
		require.NoError(err)
		require.NotNil(client)
		require.False(client.ID.IsZero())
		require.Equal("Full Metadata OIDC Client", client.ClientName)
		require.True(client.ClientURI.Valid)
		require.Equal("https://example.com", client.ClientURI.String)
		require.True(client.LogoURI.Valid)
		require.Equal("https://example.com/logo.png", client.LogoURI.String)
		require.True(client.PolicyURI.Valid)
		require.Equal("https://example.com/policy", client.PolicyURI.String)
		require.True(client.TOSURI.Valid)
		require.Equal("https://example.com/tos", client.TOSURI.String)
		require.Len(client.RedirectURIs, 2)
		require.Equal([]string{"https://example.com/callback", "https://app.example.com/cb"}, client.RedirectURIs)
		require.Len(client.Contacts, 2)
		require.Equal("admin@example.com", client.Contacts[0].String)
		require.Equal("support@example.com", client.Contacts[1].String)
		require.Equal(fullMetadataClientID, client.ClientID)
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$Bk7GvOXGHdfDdSZH1OUyIA==$1AcYMKcJwm/DngmCw9db/J7PbvPzav/i/kk+Z0EKd44=", client.Secret)
		require.False(client.CreatedBy.IsZero())
		require.False(client.Revoked.Valid)
		require.Equal(time.Date(2025, 2, 20, 21, 34, 8, 0, time.UTC), client.Created)
		require.Equal(time.Date(2025, 2, 20, 21, 34, 8, 0, time.UTC), client.Modified)

		// Check by ULID
		byID, err := s.db.RetrieveOIDCClient(s.Context(), client.ID)
		require.NoError(err)
		require.Equal(client.ID, byID.ID)
		require.Equal(client.ClientID, byID.ClientID)
		require.Equal(client.Secret, byID.Secret)
	})

	s.Run("NotFound", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), "NonExistentClientID")
		require.ErrorIs(err, errors.ErrNotFound)
		require.Nil(client)
	})

	s.Run("MissingIDEmptyString", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), "")
		require.ErrorIs(err, errors.ErrMissingID)
		require.Nil(client)
	})

	s.Run("MissingIDZeroULID", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), ulid.ULID{})
		require.ErrorIs(err, errors.ErrMissingID)
		require.Nil(client)
	})

	s.Run("InvalidType", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), 42)
		require.Error(err)
		require.Contains(err.Error(), "invalid type")
		require.Nil(client)
	})
}

func (s *storeTestSuite) TestUpdateOIDCClient() {
	if s.ReadOnly() {
		s.T().Skip("skipping update test in read-only mode")
	}

	s.Run("Success", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), fullMetadataClientID)
		require.NoError(err)
		require.NotNil(client)

		client.ClientName = "Updated Full Client Name"
		client.ClientURI = sql.NullString{Valid: true, String: "https://updated.example.com"}
		client.LogoURI = sql.NullString{Valid: true, String: "https://updated.example.com/logo.png"}
		client.PolicyURI = sql.NullString{Valid: true, String: "https://updated.example.com/policy"}
		client.TOSURI = sql.NullString{Valid: true, String: "https://updated.example.com/tos"}
		client.RedirectURIs = []string{"https://updated.example.com/cb", "https://updated.example.com/cb2"}
		client.Contacts = []sql.NullString{{Valid: true, String: "updated@example.com"}}

		err = s.db.UpdateOIDCClient(s.Context(), client)
		require.NoError(err)

		got, err := s.db.RetrieveOIDCClient(s.Context(), client.ID)
		require.NoError(err)
		require.Equal("Updated Full Client Name", got.ClientName)
		require.True(got.ClientURI.Valid)
		require.Equal("https://updated.example.com", got.ClientURI.String)
		require.True(got.LogoURI.Valid)
		require.Equal("https://updated.example.com/logo.png", got.LogoURI.String)
		require.True(got.PolicyURI.Valid)
		require.Equal("https://updated.example.com/policy", got.PolicyURI.String)
		require.True(got.TOSURI.Valid)
		require.Equal("https://updated.example.com/tos", got.TOSURI.String)
		require.Equal([]string{"https://updated.example.com/cb", "https://updated.example.com/cb2"}, got.RedirectURIs)
		require.Len(got.Contacts, 1)
		require.Equal("updated@example.com", got.Contacts[0].String)
		require.Equal(client.Secret, got.Secret)
		require.Equal(client.CreatedBy, got.CreatedBy)
		require.Equal(client.Created, got.Created)
		require.WithinDuration(time.Now(), got.Modified, 3*time.Second)
	})

	s.Run("ErrMissingID", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), fullMetadataClientID)
		require.NoError(err)
		require.NotNil(client)
		client.ID = ulid.ULID{} // non-existent ID
		err = s.db.UpdateOIDCClient(s.Context(), client)
		require.ErrorIs(err, errors.ErrMissingID)
	})

	s.Run("ErrNotFound", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), fullMetadataClientID)
		require.NoError(err)
		require.NotNil(client)
		client.ID = ulid.Make() // new ID
		err = s.db.UpdateOIDCClient(s.Context(), client)
		require.ErrorIs(err, errors.ErrNotFound)
	})

	s.Run("ValidationError", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), fullMetadataClientID)
		require.NoError(err)
		require.NotNil(client)
		client.RedirectURIs = nil // invalid: at least one redirect URI required
		err = s.db.UpdateOIDCClient(s.Context(), client)
		require.Error(err)
		require.Contains(err.Error(), "redirect_uris")
	})
}

func (s *storeTestSuite) TestRevokeOIDCClient() {
	if s.ReadOnly() {
		s.T().Skip("skipping revoke test in read-only mode")
	}

	s.Run("Success", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), fullMetadataClientID)
		require.NoError(err)
		require.NotNil(client)

		err = s.db.RevokeOIDCClient(s.Context(), client.ID)
		require.NoError(err)

		got, err := s.db.RetrieveOIDCClient(s.Context(), client.ID)
		require.NoError(err)
		require.True(got.Revoked.Valid)

		out, err := s.db.ListOIDCClients(s.Context(), nil)
		require.NoError(err)
		for _, c := range out.OIDCClients {
			require.NotEqual(fullMetadataClientID, c.ClientID, "revoked client should not appear in list")
		}
	})

	s.Run("ErrMissingID", func() {
		require := s.Require()
		err := s.db.RevokeOIDCClient(s.Context(), ulid.ULID{})
		require.ErrorIs(err, errors.ErrMissingID)
	})

	s.Run("ErrNotFound", func() {
		require := s.Require()
		err := s.db.RevokeOIDCClient(s.Context(), ulid.Make())
		require.ErrorIs(err, errors.ErrNotFound)
	})
}

func (s *storeTestSuite) TestDeleteOIDCClient() {
	if s.ReadOnly() {
		s.T().Skip("skipping delete test in read-only mode")
	}

	s.Run("Success", func() {
		require := s.Require()
		client, err := s.db.RetrieveOIDCClient(s.Context(), minimalClientID)
		require.NoError(err)
		require.NotNil(client)

		err = s.db.DeleteOIDCClient(s.Context(), client.ID)
		require.NoError(err)

		_, err = s.db.RetrieveOIDCClient(s.Context(), client.ID)
		require.ErrorIs(err, errors.ErrNotFound)
		_, err = s.db.RetrieveOIDCClient(s.Context(), minimalClientID)
		require.ErrorIs(err, errors.ErrNotFound)
	})

	s.Run("ErrMissingID", func() {
		require := s.Require()
		err := s.db.DeleteOIDCClient(s.Context(), ulid.ULID{})
		require.ErrorIs(err, errors.ErrMissingID)
	})

	s.Run("ErrNotFound", func() {
		require := s.Require()
		err := s.db.DeleteOIDCClient(s.Context(), ulid.Make())
		require.ErrorIs(err, errors.ErrNotFound)
	})
}
