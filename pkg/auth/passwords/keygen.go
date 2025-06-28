package passwords

import "go.rtnl.ai/x/randstr"

// Defaults for the length of APIKey Client IDs and Secrets
const (
	ClientIDLength     = 22
	ClientSecretLength = 48
)

// ClientID returns a random string ID that is of a fixed length with only alpha characters.
func ClientID() string {
	return randstr.Alpha(ClientIDLength)
}

// ClientSecret returns a random string of a fixed length with alpha-numeric characters.
func ClientSecret() string {
	return randstr.AlphaNumeric(ClientSecretLength)
}
