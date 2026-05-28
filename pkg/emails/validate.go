package emails

import (
	"go.rtnl.ai/commo"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

// ValidateWelcomeUserEmail renders the welcome_user templates with data to catch
// configuration errors (e.g. invalid nested welcome body) before sending.
func ValidateWelcomeUserEmail(data WelcomeUserEmailData) error {
	if data.WelcomeEmailBodyText == "" || len(data.WelcomeEmailBodyHTML) == 0 {
		return errors.ErrEmptyWelcomeEmailBody
	}
	if _, _, err := commo.Render("welcome_user", data); err != nil {
		return err
	}
	return nil
}
