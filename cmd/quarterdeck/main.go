package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"
	confire "github.com/rotationalio/confire/usage"
	"github.com/urfave/cli/v2"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/server"
	"go.rtnl.ai/quarterdeck/pkg/store"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/randstr"
)

var (
	db   store.Store
	conf config.Config
)

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
			Category: "service",
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
			Name:     "createuser",
			Usage:    "create a new user to access Quarterdeck with",
			Category: "admin",
			Before:   openDB,
			Action:   createUser,
			After:    closeDB,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "name",
					Aliases: []string{"n"},
					Usage:   "full name of user",
				},
				&cli.StringFlag{
					Name:     "email",
					Aliases:  []string{"e"},
					Required: true,
					Usage:    "email address of user",
				},
				&cli.StringSliceFlag{
					Name:    "role",
					Aliases: []string{"r"},
					Usage:   "specify the user role(s) to set their permissions (role(s) must exist in database)",
				},
			},
		},
		{
			Name:     "mkkey",
			Usage:    "generate an RSA token key pair and kid (ulid) for JWT token signing",
			Category: "admin",
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

//===========================================================================
// Admin Commands
//===========================================================================

func createUser(c *cli.Context) (err error) {
	// Lookup the role by name in the database
	var (
		roles     []*models.Role
		roleNames []string
	)

	for _, roleName := range c.StringSlice("role") {
		roleName = strings.ToLower(strings.TrimSpace(roleName))
		if roleName == "" {
			return cli.Exit("role name cannot be empty", 1)
		}

		var role *models.Role
		if role, err = db.RetrieveRole(c.Context, roleName); err != nil {
			if errors.Is(err, errors.ErrNotFound) {
				return cli.Exit(fmt.Errorf("role %q does not exist", roleName), 1)
			}
			return cli.Exit(err, 1)
		}

		roles = append(roles, role)
		roleNames = append(roleNames, role.Title)
	}

	// Assumes the user's email is verified since it is being set by an admin.
	user := &models.User{
		Name:          sql.NullString{Valid: c.String("name") != "", String: c.String("name")},
		Email:         c.String("email"),
		EmailVerified: true,
	}

	user.SetRoles(roles)

	password := randstr.AlphaNumeric(12)
	if user.Password, err = passwords.CreateDerivedKey(password); err != nil {
		return cli.Exit(err, 1)
	}

	if err = db.CreateUser(c.Context, user); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Printf("created user %s\nroles: %s\npassword: %s\n", user.Email, strings.Join(roleNames, ", "), password)
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

//===========================================================================
// Action Helpers
//===========================================================================

func openDB(c *cli.Context) (err error) {
	if conf, err = config.New(); err != nil {
		return cli.Exit(err, 1)
	}

	if db, err = store.Open(conf.Database); err != nil {
		return cli.Exit(err, 1)
	}

	return nil
}

func closeDB(c *cli.Context) error {
	if db != nil {
		if err := db.Close(); err != nil {
			return cli.Exit(err, 1)
		}
	}
	return nil
}
