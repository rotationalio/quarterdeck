package server_test

func (s *serverTestSuite) TestJWKS() {
	// Fetch the JWK resource from the well known URL
	// Ensure it returns a 200 OK status and is parseable
	// Ensure the expected keys are present in the response
	// Ensure the response has the correct headers for caching
	// Ensure the ETag header is set and matches the expected value
	// Test Last-Modified header is set and matches the expected value
	// Ensure no Expires header is returned in response
	// Modify the JWKS and ensure that the ETag and last-modified change
	// Ensure that If-None-Match returns 304
	// Ensure that If-Modified-Since returns 304
	// Ensure the Cache Control Directives are correct
}

func (s *serverTestSuite) TestOpenIDConfiguration() {
	// Fetch the OpenID configuration from the well known URL
	// Ensure it returns a 200 OK status and is parseable
	// Ensure the expected fields are present in the response
	// Ensure the issuer URL is correctly formed
	// Ensure the JWKS URI is correctly formed
	// Ensure the response has the correct headers for caching
}

func (s *serverTestSuite) TestSecurityTxt() {
	// Fetch the security.txt file from the well known URL
	// Ensure it returns a 200 OK status and is parseable
	// Ensure the response has the correct headers for caching
	// Ensure the content type is set to text/plain
	// Ensure the key exists and can be fetched
	// Ensure the content can be verified with the public key
}
