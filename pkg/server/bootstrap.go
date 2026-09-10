package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v1/models"
	"go.rtnl.ai/x/rlog"
)

// BootstrapTimeout is the timeout for the bootstrap process.
const BootstrapTimeout = 30 * time.Second

// Runs the bootstrap process for the server.
func (s *Server) Bootstrap() (err error) {
	// Do not continue if in maintenance mode or bootstrap is not enabled.
	if s.conf.Maintenance || !s.conf.Bootstrap.Enabled {
		rlog.Debug(
			"bootstrap is not enabled",
			slog.Bool("maintenance", s.conf.Maintenance),
			slog.Bool("enabled", s.conf.Bootstrap.Enabled))
		return nil
	}

	// If bootstrap config is invalid, stop bootstrap processing.
	if err = s.conf.Bootstrap.Validate(); err != nil {
		rlog.Error("cannot bootstrap server", slog.String("error", err.Error()))
		return fmt.Errorf("cannot bootstrap server: %w", err)
	}

	// Create a new context for the bootstrap process.
	ctx, cancel := context.WithTimeout(context.Background(), BootstrapTimeout)
	defer cancel()

	// Attempt to create the superuser if configured.
	if s.conf.Bootstrap.Superuser.Enabled() {
		if err = s.BootstrapSuperuser(ctx); err != nil {
			rlog.Error("cannot bootstrap superuser", slog.String("error", err.Error()))
			return fmt.Errorf("cannot bootstrap superuser: %w", err)
		}
	}

	rlog.Debug("bootstrap completed")
	return nil
}

func (s *Server) BootstrapSuperuser(ctx context.Context) (err error) {
	rlog.Debug(
		"bootstrapping superuser",
		slog.String("name", s.conf.Bootstrap.Superuser.Name),
		slog.String("email", s.conf.Bootstrap.Superuser.Email),
	)

	var (
		created, verified bool
		superuser         *models.User
	)

	// Create the super user if it does not exist, otherwise verify the password.
	if superuser, err = s.store.RetrieveUser(ctx, s.conf.Bootstrap.Superuser.Email); err != nil {
		if !errors.Is(err, errors.ErrNotFound) {
			return err
		}

		// Create the superuser.
		superuser = &models.User{
			Name:          sql.NullString{Valid: s.conf.Bootstrap.Superuser.Name != "", String: s.conf.Bootstrap.Superuser.Name},
			Email:         s.conf.Bootstrap.Superuser.Email,
			EmailVerified: true,
		}

		// Create the derived key for the password
		if superuser.Password, err = passwords.CreateDerivedKey(s.conf.Bootstrap.Superuser.Password); err != nil {
			return fmt.Errorf("could not create derived key for superuser password: %w", err)
		}

		// Create the superuser
		if err = s.store.CreateUser(ctx, superuser); err != nil {
			return fmt.Errorf("could not create superuser: %w", err)
		}

		created = true
	} else {
		if verified, err = passwords.VerifyDerivedKey(superuser.Password, s.conf.Bootstrap.Superuser.Password); err != nil {
			return fmt.Errorf("could not verify superuser password: %w", err)
		}

		// If the password is incorrect but force password is true, update the password.
		if !verified {
			if s.conf.Bootstrap.Superuser.ForcePassword {
				// Update the password
				var derivedKey string
				if derivedKey, err = passwords.CreateDerivedKey(s.conf.Bootstrap.Superuser.Password); err != nil {
					return fmt.Errorf("could not create derived key for superuser password: %w", err)
				}

				// Update the password
				if err = s.store.UpdatePassword(ctx, superuser.ID, derivedKey); err != nil {
					return fmt.Errorf("could not update superuser password: %w", err)
				}

				// Update the verified flag since we forced the password update
				verified = true
			} else {
				// Otherwise log and return the error
				return errors.ErrFailedAuthentication
			}
		}
	}

	rlog.Info(
		"superuser bootstrap completed",
		slog.String("name", s.conf.Bootstrap.Superuser.Name),
		slog.String("email", s.conf.Bootstrap.Superuser.Email),
		slog.Bool("created", created),
		slog.Bool("verified", verified),
		slog.Bool("force_password", s.conf.Bootstrap.Superuser.ForcePassword),
	)
	return nil
}
