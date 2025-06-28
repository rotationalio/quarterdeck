package main

import (
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/urfave/cli/v2"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/vero"
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
			Category:  "testing",
			Action:    derkey,
			ArgsUsage: "password [password ...]",
			Flags:     []cli.Flag{},
		},
		{
			Name:     "keypair",
			Usage:    "create a fake apikey client ID and secret to use as a fixture for testing",
			Category: "testing",
			Action:   keypair,
			Flags:    []cli.Flag{},
		},
		{
			Name:     "mkkey",
			Usage:    "generate an RSA token key pair and kid (ulid) for JWT token signing",
			Category: "testing",
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
		{
			Name:     "vero",
			Usage:    "generate a vero token serialized as a database input for testing",
			Category: "testing",
			Action:   veroToken,
			Flags:    []cli.Flag{},
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

	// Generate Signing Key Pair using Signing Key algorithm currently in use.
	var keypair auth.SigningKey
	if keypair, err = auth.GenerateKeys(); err != nil {
		return cli.Exit(err, 1)
	}

	// Save the keypair to the specified file in PEM format.
	if err = keypair.Dump(out); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Printf("signing key id: %s -- saved with PEM encoding to %s\n", keyid, out)
	return nil
}

func veroToken(c *cli.Context) (err error) {
	resourceID := ulid.MakeSecure()
	expiration := time.Now().Add(87600 * time.Hour)

	var token *vero.Token
	if token, err = vero.New(resourceID[:], expiration); err != nil {
		return cli.Exit(err, 1)
	}

	var signature *vero.SignedToken
	if _, signature, err = token.Sign(); err != nil {
		return cli.Exit(err, 1)
	}

	var value driver.Value
	if value, err = signature.Value(); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Println(value)
	return nil
}
