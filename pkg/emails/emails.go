package emails

import (
	"context"
	"errors"
	"time"

	"github.com/jordan-wright/email"
	"github.com/rs/zerolog/log"
	"go.rtnl.ai/x/backoff"

	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go"
	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Package level variable to enclose email sending details.
var (
	initialized bool
	config      Config
	pool        *email.Pool
	sg          *sendgrid.Client
)

// Hidden package level variables for sending messages.
const (
	defaultTimeout      = 30 * time.Second
	multiplier          = 2.0
	randomizationFactor = 0.45
	maxInterval         = 45 * time.Second
	maxElapsedTime      = 180 * time.Second
	initialInterval     = 2500 * time.Millisecond
)

// Configure the package to start sending emails. If there is no valid email
// configuration available then configuration is gracefully ignored without error.
func Configure(conf Config) (err error) {
	// Do not configure email if it is not available but also do not return an error.
	if !conf.Available() {
		return nil
	}

	if err = conf.Validate(); err != nil {
		return err
	}

	// TODO: if in testing mode create a mock for sending emails.

	if conf.SMTP.Enabled() {
		if pool, err = conf.SMTP.Pool(); err != nil {
			return err
		}
	}

	if conf.SendGrid.Enabled() {
		sg = conf.SendGrid.Client()
	}

	config = conf
	initialized = true
	return nil
}

// Send an email using the configured send methodology. Uses exponential backoff to
// retry multiple times on error with an increasing delay between attempts.
func Send(email *Email) (err error) {
	// The package must be initialized to send.
	if !initialized {
		return ErrNotInitialized
	}

	// Select the send function to deliver the email with.
	var send sender
	switch {
	case config.SMTP.Enabled():
		send = sendSMTP
	case config.SendGrid.Enabled():
		send = sendSendGrid
	case config.Testing:
		send = sendMock
	default:
		panic("unhandled send email method")
	}

	// Attempt to send the message with multiple retries using exponential backoff.
	exponential := backoff.ExponentialBackOff{
		InitialInterval:     initialInterval,
		RandomizationFactor: randomizationFactor,
		Multiplier:          multiplier,
		MaxInterval:         maxInterval,
	}

	if _, err = backoff.Retry(context.Background(), func() (any, serr error) {
		serr = send(email)
		return nil, serr
	},
		backoff.WithBackOff(&exponential),
		backoff.WithMaxElapsedTime(maxElapsedTime),
		backoff.WithNotify(func(err error, delay time.Duration) {
			log.Debug().Err(err).Dur("delay", delay).Msg("could not send email, retrying after delay")
		}),
	); err != nil {
		return err
	}

	return nil

}

type sender func(*Email) error

func sendSMTP(e *Email) (err error) {
	var msg *email.Email
	if msg, err = e.ToSMTP(); err != nil {
		return err
	}

	if err = pool.Send(msg, defaultTimeout); err != nil {
		return err
	}
	return nil
}

func sendSendGrid(e *Email) (err error) {
	var msg *sgmail.SGMailV3
	if msg, err = e.ToSendGrid(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var rep *rest.Response
	if rep, err = sg.SendWithContext(ctx, msg); err != nil {
		return err
	}

	if rep.StatusCode < 200 || rep.StatusCode >= 300 {
		return errors.New(rep.Body)
	}

	return nil
}

func sendMock(*Email) (err error) {
	return errors.New("not implemented")
}
