package config

type DatabaseConfig struct {
	URL      string `default:"sqlite3:////data/db/quarterdeck.db" desc:"the database connection URL, including the driver to use."`
	ReadOnly bool   `split_words:"true" default:"false" desc:"if true, quarterdeck will not write to the database, only read from it"`
}
