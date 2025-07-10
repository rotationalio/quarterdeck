package sqlite_test

func (s *storeTestSuite) TestAPIKeyList() {
	require := s.Require()
	out, err := s.db.ListAPIKeys(s.Context(), nil)
	require.NoError(err, "should be able to list api keys")
	require.NotNil(out, "should return an api key list")
	require.Len(out.APIKeys, 3, "api key list should return 3 keys and none that are revoked")

	// Ensure no keys returned are revoked
	for _, key := range out.APIKeys {
		require.False(key.Revoked.Valid, "api key should not be revoked")
	}
}
