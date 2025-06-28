package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/ulid"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/store/mock"
	. "go.rtnl.ai/quarterdeck/pkg/store/models"
)

func TestVeroParams(t *testing.T) {
	// This test ensures that vero params are correctly returned in the correct orde
	// to prevent developer typos that may lead to hard to find bugs. It's annoying
	// because anytime you add a new field, you have to update this test, but it
	// will prevent headaches for you later on, I promise.
	vero := &VeroToken{
		Model: Model{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		TokenType: enum.TokenTypeResetPassword,
	}

	CheckParams(t, vero.Params(),
		[]string{
			"id", "tokenType", "resourceID", "email", "expiration", "signature", "sentOn", "created", "modified",
		},
		[]any{
			vero.ID, vero.TokenType, vero.ResourceID, vero.Email, vero.Expiration, vero.Signature, vero.SentOn, vero.Created, vero.Modified,
		},
	)
}

func TestVeroScan(t *testing.T) {
	t.Run("NotNull", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(),    // ID
			"verify_email",                // TokenType
			ulid.MakeSecure().String(),    // ResourceID
			"gdavies@example.com",         // Email
			time.Now().Add(5 * time.Hour), // Expiration
			[]byte{1, 151, 177, 135, 47, 223, 130, 157, 127, 242, 128, 127, 227, 101, 35, 22, 240, 132, 190, 169, 154, 183, 165, 173, 57, 118, 185, 132, 29, 203, 211, 168, 124, 90, 31, 213, 27, 248, 40, 121, 14, 77, 32, 181, 246, 249, 62, 29, 198, 177, 155, 26, 116, 10, 166, 157, 97, 246, 236, 118, 118, 115, 153, 231, 223, 15, 14, 116, 217, 70, 7, 173, 160, 173, 45, 105, 246, 206, 228, 34, 99, 182, 240, 208, 148, 77, 182, 85, 57, 207, 18, 56, 55, 248, 8, 136, 41, 191, 29, 132, 155, 17, 84, 169, 47, 240, 196, 8, 96, 122, 188, 6, 73, 116, 137, 102, 209, 42, 201, 110, 101}, // Signature
			time.Now().Add(-1 * time.Hour), // SentOn
			time.Now().Add(-1 * time.Hour), // Created
			time.Now().Add(-1 * time.Hour), // Modified
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		model := &VeroToken{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, enum.TokenTypeVerifyEmail, model.TokenType, "expected field TokenType to match data[1]")
		require.Equal(t, ulid.MustParse(data[2]), model.ResourceID.ULID, "expected field ResourceID to match data[2]")
		require.Equal(t, data[3], model.Email, "expected field Email to match data[3]")
		require.Equal(t, data[4], model.Expiration, "expected field Expiration to match data[4]")
		require.NotNil(t, model.Signature, "expected field Signature to be not nil")
		require.NotEmpty(t, model.Signature.RecordID, "expected field Signature to be populated with a valid SignedToken")
		require.Equal(t, data[6], model.SentOn.Time, "expected field SentOn to match data[6]")
		require.Equal(t, data[7], model.Created, "expected field Created to match data[7]")
		require.Equal(t, data[8], model.Modified, "expected field Modified to match data[8]")
	})

	t.Run("Nulls", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(),     // ID
			"verify_email",                 // TokenType
			nil,                            // ResourceID
			"gdavies@example.com",          // Email
			time.Now().Add(5 * time.Hour),  // Expiration
			nil,                            // Signature
			nil,                            // SentOn
			time.Now().Add(-1 * time.Hour), // Created
			time.Time{},                    // Modified
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		//test
		model := &VeroToken{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.False(t, model.ResourceID.Valid, "expected field ResourceID to be invalid (null)")
		require.False(t, model.SentOn.Valid, "expected field SentOn to be invalid (null)")
		require.Nil(t, model.Signature, "expected field Signature to be nil")
		require.True(t, model.Modified.IsZero(), "expected field Modified to be zero time")
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &VeroToken{}
		err := model.Scan(mockScanner)
		require.ErrorIs(t, err, ErrModelScan, "expected error when scanning with mock scanner")
	})
}
