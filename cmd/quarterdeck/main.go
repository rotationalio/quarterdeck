package main

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/joho/godotenv"
	confire "github.com/rotationalio/confire/usage"
	"github.com/urfave/cli/v2"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/server"
	"go.rtnl.ai/ulid"
)

var conf config.Config

func main() {
	// If a dotenv file exists, load it for configuration
	godotenv.Load()

	// Create a multi-command CLI application
	app := cli.NewApp()
	app.Name = "quarterdeck"
	app.Version = pkg.Version(false)
	app.Usage = "run and manage quarterdeck services"
	app.Flags = []cli.Flag{}
	app.Commands = []*cli.Command{
		{
			Name:     "serve",
			Usage:    "run the quarterdeck server",
			Action:   serve,
			Category: "service",
			Flags:    []cli.Flag{},
		},
		{
			Name:     "config",
			Usage:    "print quarterdeck configuration guide",
			Category: "utility",
			Action:   usage,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "list",
					Aliases: []string{"l"},
					Usage:   "print in list mode instead of table mode",
				},
			},
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
// Server Commands
//===========================================================================

func serve(c *cli.Context) (err error) {
	if conf, err = config.New(); err != nil {
		return cli.Exit(err, 1)
	}

	var srv *server.Server
	if srv, err = server.New(conf); err != nil {
		return cli.Exit(err, 1)
	}

	if err = srv.Serve(); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

//===========================================================================
// Utility Commands
//===========================================================================

func usage(c *cli.Context) (err error) {
	tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
	format := confire.DefaultTableFormat
	if c.Bool("list") {
		format = confire.DefaultListFormat
	}

	var conf config.Config
	if err := confire.Usagef("quarterdeck", &conf, tabs, format); err != nil {
		return cli.Exit(err, 1)
	}
	tabs.Flush()
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
