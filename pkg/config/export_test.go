package config

// Export internal methods for testing only.
var (
	Reset = reset
)

func GlobalConf() *Config {
	return conf
}

func GlobalConfErr() error {
	return confErr
}
