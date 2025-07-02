package main

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"
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
		{
			Name:      "fixture",
			Usage:     "generate a fixture stub for adding to database test cases",
			Args:      true,
			UsageText: "bosun fixture [ulid|int|modified|email|time|t|f|_|'string'|d|b'blob' ...]",
			Category:  "testing",
			Action:    fixtureStub,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "model",
					Aliases: []string{"m"},
					Usage:   "name of the model to generate a fixture stub for",
					Value:   "",
				},
				&cli.TimestampFlag{
					Name:   "epoch",
					Usage:  "the date/time the database was created, to generate fixture timestamps",
					Layout: time.RFC3339,
					Value:  cli.NewTimestamp(time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC)),
				},
				&cli.DurationFlag{
					Name:    "age",
					Aliases: []string{"a"},
					Usage:   "the age of the database to use for generating timestamps in the fixture stub",
					Value:   time.Hour * 24 * 120, // Default to three months
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

var isDigit = regexp.MustCompile(`^\d+$`)

func fixtureStub(c *cli.Context) error {
	params := make([]string, 0, c.NArg()+2)
	created, modified := auditTimes(c)

	for i := 0; i < c.NArg(); i++ {
		field := c.Args().Get(i)
		switch field {
		case "ulid":
			uu, _ := ulid.New(ulid.Timestamp(created), ulid.DefaultEntropy())
			params = append(params, fmt.Sprintf("x'%s'", hex.EncodeToString(uu[:])))
		case "int":
			params = append(params, fmt.Sprintf("'%d'", rand.IntN(1000)))
		case "modified":
			params = append(params, fmt.Sprintf("'%s'", modified.Format(time.RFC3339)))
		case "email":
			params = append(params, "'@example.com'")
		case "time", "ts", "timestamp":
			ts := timeInRange(created, modified)
			params = append(params, fmt.Sprintf("'%s'", ts.Format(time.RFC3339)))
		case "t", "true":
			params = append(params, "'t'")
		case "f", "false":
			params = append(params, "'f'")
		case "b", "blank", "_":
			params = append(params, "''")
		case "N", "null", "NULL", "nil":
			params = append(params, "NULL")
		default:
			// If it is a number write it directly; if it is a string write it as a
			// quoted string. If it is prefixed with a b' then treat it as a blob.
			if strings.HasPrefix(field, "b'") {
				// Remove the b' prefix and the trailing ' to get the hex value.
				blob := strings.TrimSuffix(strings.TrimPrefix(field, "b'"), "'")
				params = append(params, fmt.Sprintf("x'%s'", hex.EncodeToString([]byte(blob))))
				continue
			}

			if strings.HasPrefix(field, "x'") {
				// Write an x field exactly as it is, assuming it is a hex value.
				params = append(params, field)
				continue
			}

			if isDigit.MatchString(field) {
				params = append(params, field)
				continue
			}

			field = strings.Trim(field, `'"`)
			params = append(params, fmt.Sprintf("'%s'", field))
		}
	}

	params = append(params, fmt.Sprintf("'%s'", created.Format(time.RFC3339)))
	params = append(params, fmt.Sprintf("'%s'", modified.Format(time.RFC3339)))

	fmt.Printf("(%s),\n", strings.Join(params, ", "))
	return nil
}

func auditTimes(c *cli.Context) (created, modified time.Time) {
	var epochs time.Time
	if ts := c.Timestamp("epoch"); ts != nil {
		epochs = *ts
	} else {
		// Default to a fixed date if not provided
		epochs = time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC)
	}

	epoche := epochs.Add(c.Duration("age"))

	created = timeInRange(epochs, epoche)
	modified = timeInRange(created, epoche)

	return created, modified
}

func timeInRange(start, end time.Time) time.Time {
	duration := end.Sub(start)
	increase := time.Duration(rand.Int64N(int64(duration)))
	return start.Add(increase)
}
