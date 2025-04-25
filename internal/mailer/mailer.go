package mailer

import "embed"

const (
	FromName               = "BMS"
	maxRetires             = 3
	UserInvitationTemplate = "user_invitation.tmpl"
	WelcomeTemplate        = "welcome.tmpl"
)

//go:embed "templates"
var FS embed.FS

type Client interface {
	Send(templateFile, username, email string, data any) (int, error)
}
