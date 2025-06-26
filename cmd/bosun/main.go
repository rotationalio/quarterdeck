package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/urfave/cli/v2"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/ulid"
)

func main() {
	// If a dotenv file exists, load it for configuration
	godotenv.Load()

	// Create a multi-command CLI application
	app := cli.NewApp()
	app.Name = "bosun"
	app.Version = pkg.Version(false)
	app.Usage = "helpers for quarterdeck testing, debugging, and code generation"
	app.Flags = []cli.Flag{}
	app.Commands = []*cli.Command{
		{
			Name:      "argon2",
			Usage:     "create a derived key to use as a fixture for testing",
			Category:  "debug",
			Action:    derkey,
			ArgsUsage: "password [password ...]",
			Flags:     []cli.Flag{},
		},
		{
			Name:     "keypair",
			Usage:    "create a fake apikey client ID and secret to use as a fixture for testing",
			Category: "debug",
			Action:   keypair,
			Flags:    []cli.Flag{},
		},
		{
			Name:     "mkkey",
			Usage:    "generate an RSA token key pair and kid (ulid) for JWT token signing",
			Category: "utility",
			Action:   mkkey,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "out",
					Aliases: []string{"o"},
					Usage:   "path to write keys out to (optional, will be saved as [kid].pem by default)",
				},
				&cli.IntFlag{
					Name:    "size",
					Aliases: []string{"s"},
					Usage:   "number of bits for the generated keys",
					Value:   4096,
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

//===========================================================================
// Commands
//===========================================================================

func derkey(c *cli.Context) error {
	if c.NArg() == 0 {
		return cli.Exit("specify password(s) to create argon2 derived key(s) from", 1)
	}

	for i := 0; i < c.NArg(); i++ {
		pwdk, err := passwords.CreateDerivedKey(c.Args().Get(i))
		if err != nil {
			return cli.Exit(err, 1)
		}
		fmt.Println(pwdk)
	}

	return nil
}

func keypair(c *cli.Context) error {
	clientID := passwords.ClientID()
	secret := passwords.ClientSecret()
	fmt.Printf("%s.%s\n", clientID, secret)
	return nil
}

func mkkey(c *cli.Context) (err error) {
	// Create ULID and determine outpath
	keyid := ulid.Make()

	var out string
	if out = c.String("out"); out == "" {
		out = fmt.Sprintf("%s.pem", keyid)
	}

	// Generate RSA keys using crypto random
	var key *rsa.PrivateKey
	if key, err = rsa.GenerateKey(rand.Reader, c.Int("size")); err != nil {
		return cli.Exit(err, 1)
	}

	// Open file to PEM encode keys to
	var f *os.File
	if f, err = os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600); err != nil {
		return cli.Exit(err, 1)
	}

	if err = pem.Encode(f, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Printf("RSA key id: %s -- saved with PEM encoding to %s\n", keyid, out)
	return nil
}
