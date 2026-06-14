package db_test

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal/fields"
	"go.rtnl.ai/ulid"
)

const (
	// Fixture client IDs from suitetest/testdata/0004_oidc_clients.sql.
	fullMetadataClientID = "OidcClient1FullMetadata"
	minimalClientID      = "OidcClient2Minimal"
	keyholderUserULID    = "01JPYRNYMEHNEZCS0JYX1CP57A"
)

//=============================================================================
// OIDC Client Store Tests
//=============================================================================

// TestListOIDCClients verifies list cursor returns full and minimal metadata clients.
func (s *storeSuite) TestListOIDCClients() {
	s.Run("Maximal", func() {
		require := s.Require()
		// Action: list all OIDC clients.
		cursor, err := s.store.ListOIDCClients(s.Context(), nil)
		require.NoError(err)
		defer func() { require.NoError(cursor.Close()) }()
		clients, err := cursor.List()
		require.NoError(err)
		require.Len(clients, 2)

		// Assert: full-metadata fixture client has all optional fields populated.
		var full *models.OIDCClient
		for _, c := range clients {
			if c.ClientID == fullMetadataClientID {
				full = c
				break
			}
		}
		require.NotNil(full)
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
		require.Equal(fields.StringArray{"https://example.com/callback", "https://app.example.com/cb"}, full.RedirectURIs)
		require.Len(full.Contacts, 2)
		require.Equal("admin@example.com", full.Contacts[0])
		require.Equal("support@example.com", full.Contacts[1])
		require.Equal(fullMetadataClientID, full.ClientID)
		require.Empty(full.Secret)
		require.False(full.CreatedBy.IsZero())
		require.False(full.Created.IsZero())
		require.False(full.Modified.IsZero())
	})

	s.Run("Minimal", func() {
		require := s.Require()
		// Action: list all OIDC clients.
		cursor, err := s.store.ListOIDCClients(s.Context(), nil)
		require.NoError(err)
		defer func() { require.NoError(cursor.Close()) }()
		clients, err := cursor.List()
		require.NoError(err)

		// Assert: minimal-metadata fixture client omits optional fields.
		var minimal *models.OIDCClient
		for _, c := range clients {
			if c.ClientID == minimalClientID {
				minimal = c
				break
			}
		}
		require.NotNil(minimal)
		require.Equal("Minimal Metadata OIDC Client", minimal.ClientName)
		require.False(minimal.ClientURI.Valid)
		require.False(minimal.LogoURI.Valid)
		require.False(minimal.PolicyURI.Valid)
		require.False(minimal.TOSURI.Valid)
		require.NotNil(minimal.RedirectURIs)
		require.Len(minimal.RedirectURIs, 1)
		require.Equal("https://example.com/cb", minimal.RedirectURIs[0])
		require.Empty(minimal.Contacts)
		require.Equal(minimalClientID, minimal.ClientID)
		require.Empty(minimal.Secret)
	})
}

// TestCreateOIDCClient verifies validation and creation with maximal/minimal metadata.
func (s *storeSuite) TestCreateOIDCClient() {
	s.Run("NoIDOnCreate", func() {
		client := &models.OIDCClient{
			ClientName: "Test",
			ClientID:   "TestClientID123",
			Secret:     "secret",
			CreatedBy:  ulid.MustParse(keyholderUserULID),
		}
		client.ID = ulid.Make()

		_, err := s.store.CreateOIDCClient(s.Context(), client)
		s.Require().ErrorIs(err, errors.ErrNoIDOnCreate)
	})

	s.Run("RejectsEmptyClientID", func() {
		client := &models.OIDCClient{
			ClientName:   "Test",
			ClientID:     "",
			Secret:       "secret",
			CreatedBy:    ulid.MustParse(keyholderUserULID),
			RedirectURIs: fields.StringArray{"https://example.com/cb"},
		}
		_, err := s.store.CreateOIDCClient(s.Context(), client)
		s.Require().ErrorIs(err, errors.ErrZeroValuedNotNull)
	})

	s.Run("RejectsEmptySecret", func() {
		client := &models.OIDCClient{
			ClientName:   "Test",
			ClientID:     "valid-client-id",
			Secret:       "",
			CreatedBy:    ulid.MustParse(keyholderUserULID),
			RedirectURIs: fields.StringArray{"https://example.com/cb"},
		}
		_, err := s.store.CreateOIDCClient(s.Context(), client)
		s.Require().ErrorIs(err, errors.ErrZeroValuedNotNull)
	})

	s.Run("Maximal", func() {
		require := s.Require()
		// Setup: client with all optional metadata fields populated.
		client := &models.OIDCClient{
			ClientName:   "Created Full Client",
			ClientURI:    sql.NullString{Valid: true, String: "https://created.example.com"},
			LogoURI:      sql.NullString{Valid: true, String: "https://created.example.com/logo.png"},
			PolicyURI:    sql.NullString{Valid: true, String: "https://created.example.com/policy"},
			TOSURI:       sql.NullString{Valid: true, String: "https://created.example.com/tos"},
			RedirectURIs: fields.StringArray{"https://created.example.com/cb", "https://created.example.com/cb2"},
			Contacts:     fields.StringArray{"created@example.com", "support@created.example.com"},
			ClientID:     "CreatedFullOIDCClient",
			Secret:       "$argon2id$v=19$m=65536,t=1,p=2$createdsecretbase64$createdsaltsuffix",
			CreatedBy:    ulid.MustParse(keyholderUserULID),
		}
		// Action: create client and reload by client ID.
		created, err := s.store.CreateOIDCClient(s.Context(), client)
		require.NoError(err)
		require.False(created.ID.IsZero())
		require.WithinDuration(time.Now(), created.Created, 3*time.Second)
		require.WithinDuration(time.Now(), created.Modified, 3*time.Second)

		got, err := s.store.RetrieveOIDCClientByClientID(s.Context(), client.ClientID)
		require.NoError(err)
		// Assert: all metadata fields round-trip unchanged.
		require.Equal(created.ID, got.ID)
		require.Equal(client.ClientName, got.ClientName)
		require.Equal(client.ClientURI, got.ClientURI)
		require.Equal(client.LogoURI, got.LogoURI)
		require.Equal(client.PolicyURI, got.PolicyURI)
		require.Equal(client.TOSURI, got.TOSURI)
		require.Equal(client.RedirectURIs, got.RedirectURIs)
		require.Len(got.Contacts, 2)
		require.Equal(client.Contacts[0], got.Contacts[0])
		require.Equal(client.Contacts[1], got.Contacts[1])
		require.Equal(client.ClientID, got.ClientID)
		require.Equal(client.Secret, got.Secret)
		require.Equal(client.CreatedBy, got.CreatedBy)
		require.WithinDuration(created.Created, got.Created, time.Second)
		require.WithinDuration(created.Modified, got.Modified, time.Second)
	})

	s.Run("Minimal", func() {
		require := s.Require()
		// Setup: client with only required fields populated.
		client := &models.OIDCClient{
			ClientName:   "",
			ClientURI:    sql.NullString{Valid: false},
			LogoURI:      sql.NullString{Valid: false},
			PolicyURI:    sql.NullString{Valid: false},
			TOSURI:       sql.NullString{Valid: false},
			RedirectURIs: fields.StringArray{"https://minimal.example.com/cb"},
			Contacts:     nil,
			ClientID:     "CreatedMinimalOIDCClient",
			Secret:       "$argon2id$v=19$m=65536,t=1,p=2$minimalsecret$minimalsalt",
			CreatedBy:    ulid.MustParse(keyholderUserULID),
		}
		// Action: create client and reload by client ID.
		created, err := s.store.CreateOIDCClient(s.Context(), client)
		require.NoError(err)
		require.False(created.ID.IsZero())

		got, err := s.store.RetrieveOIDCClientByClientID(s.Context(), client.ClientID)
		require.NoError(err)
		// Assert: optional metadata fields remain unset.
		require.Equal(client.ClientName, got.ClientName)
		require.False(got.ClientURI.Valid)
		require.False(got.LogoURI.Valid)
		require.False(got.PolicyURI.Valid)
		require.False(got.TOSURI.Valid)
		require.Equal(client.RedirectURIs, got.RedirectURIs)
		require.Empty(got.Contacts)
		require.Equal(client.ClientID, got.ClientID)
		require.Equal(client.Secret, got.Secret)
	})
}

// TestRetrieveOIDCClient verifies lookup by client ID, ULID, and not-found cases.
func (s *storeSuite) TestRetrieveOIDCClient() {
	s.Run("Success", func() {
		require := s.Require()
		client, err := s.store.RetrieveOIDCClientByClientID(s.Context(), fullMetadataClientID)
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
		require.Equal(fields.StringArray{"https://example.com/callback", "https://app.example.com/cb"}, client.RedirectURIs)
		require.Len(client.Contacts, 2)
		require.Equal("admin@example.com", client.Contacts[0])
		require.Equal("support@example.com", client.Contacts[1])
		require.Equal(fullMetadataClientID, client.ClientID)
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$Bk7GvOXGHdfDdSZH1OUyIA==$1AcYMKcJwm/DngmCw9db/J7PbvPzav/i/kk+Z0EKd44=", client.Secret)
		require.False(client.CreatedBy.IsZero())
		require.Equal(time.Date(2025, 2, 20, 21, 34, 8, 0, time.UTC), client.Created)
		require.Equal(time.Date(2025, 2, 20, 21, 34, 8, 0, time.UTC), client.Modified)

		byID, err := s.store.RetrieveOIDCClient(s.Context(), client.ID)
		require.NoError(err)
		require.Equal(client.ID, byID.ID)
		require.Equal(client.ClientID, byID.ClientID)
		require.Equal(client.Secret, byID.Secret)
	})

	s.Run("NotFound", func() {
		require := s.Require()
		client, err := s.store.RetrieveOIDCClientByClientID(s.Context(), "NonExistentClientID")
		require.ErrorIs(err, errors.ErrNotFound)
		require.Nil(client)
	})

	s.Run("MissingIDEmptyString", func() {
		require := s.Require()
		client, err := s.store.RetrieveOIDCClientByClientID(s.Context(), "")
		require.ErrorIs(err, errors.ErrNotFound)
		require.Nil(client)
	})

	s.Run("MissingIDZeroULID", func() {
		require := s.Require()
		client, err := s.store.RetrieveOIDCClient(s.Context(), ulid.ULID{})
		require.ErrorIs(err, errors.ErrNotFound)
		require.Nil(client)
	})
}

// TestUpdateOIDCClient verifies metadata updates and error handling for missing/unknown IDs.
func (s *storeSuite) TestUpdateOIDCClient() {
	s.Run("Success", func() {
		require := s.Require()
		client, err := s.store.RetrieveOIDCClientByClientID(s.Context(), fullMetadataClientID)
		require.NoError(err)
		require.NotNil(client)

		client.ClientName = "Updated Full Client Name"
		client.ClientURI = sql.NullString{Valid: true, String: "https://updated.example.com"}
		client.LogoURI = sql.NullString{Valid: true, String: "https://updated.example.com/logo.png"}
		client.PolicyURI = sql.NullString{Valid: true, String: "https://updated.example.com/policy"}
		client.TOSURI = sql.NullString{Valid: true, String: "https://updated.example.com/tos"}
		client.RedirectURIs = fields.StringArray{"https://updated.example.com/cb", "https://updated.example.com/cb2"}
		client.Contacts = fields.StringArray{"updated@example.com"}

		err = s.store.UpdateOIDCClient(s.Context(), client)
		require.NoError(err)

		got, err := s.store.RetrieveOIDCClient(s.Context(), client.ID)
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
		require.Equal(fields.StringArray{"https://updated.example.com/cb", "https://updated.example.com/cb2"}, got.RedirectURIs)
		require.Len(got.Contacts, 1)
		require.Equal("updated@example.com", got.Contacts[0])
		require.Equal(client.Secret, got.Secret)
		require.Equal(client.CreatedBy, got.CreatedBy)
		require.Equal(client.Created, got.Created)
		require.WithinDuration(time.Now(), got.Modified, 3*time.Second)
	})

	s.Run("ErrMissingID", func() {
		require := s.Require()
		client, err := s.store.RetrieveOIDCClientByClientID(s.Context(), fullMetadataClientID)
		require.NoError(err)
		require.NotNil(client)
		client.ID = ulid.ULID{}
		err = s.store.UpdateOIDCClient(s.Context(), client)
		require.ErrorIs(err, errors.ErrMissingID)
	})

	s.Run("ErrNotFound", func() {
		require := s.Require()
		client, err := s.store.RetrieveOIDCClientByClientID(s.Context(), fullMetadataClientID)
		require.NoError(err)
		require.NotNil(client)
		client.ID = ulid.Make()
		err = s.store.UpdateOIDCClient(s.Context(), client)
		require.ErrorIs(err, errors.ErrNotFound)
	})
}

// TestDeleteOIDCClient verifies deletion and error handling for missing/unknown IDs.
func (s *storeSuite) TestDeleteOIDCClient() {
	s.Run("Success", func() {
		require := s.Require()
		client, err := s.store.RetrieveOIDCClientByClientID(s.Context(), minimalClientID)
		require.NoError(err)
		require.NotNil(client)

		err = s.store.DeleteOIDCClient(s.Context(), client.ID)
		require.NoError(err)

		_, err = s.store.RetrieveOIDCClient(s.Context(), client.ID)
		require.ErrorIs(err, errors.ErrNotFound)
		_, err = s.store.RetrieveOIDCClientByClientID(s.Context(), minimalClientID)
		require.ErrorIs(err, errors.ErrNotFound)
	})

	s.Run("ErrMissingID", func() {
		require := s.Require()
		err := s.store.DeleteOIDCClient(s.Context(), ulid.ULID{})
		require.ErrorIs(err, errors.ErrMissingID)
	})

	s.Run("ErrNotFound", func() {
		require := s.Require()
		err := s.store.DeleteOIDCClient(s.Context(), ulid.Make())
		require.ErrorIs(err, errors.ErrNotFound)
	})
}
