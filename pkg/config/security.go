package config

type SecurityConfig struct {
	TxtPath string `split_words:"true" required:"false" desc:"path to the security.txt file to serve at /.well-known/security.txt"`
}
