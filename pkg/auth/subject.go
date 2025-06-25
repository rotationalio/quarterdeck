package auth

type SubjectType rune

const (
	SubjectUser   = SubjectType('u') // User subject type.
	SubjectAPIKey = SubjectType('k') // API key subject type.
	SubjectVero   = SubjectType('v') // Used for Vero (e.g. password reset, email verification).
)

func (s SubjectType) String() string {
	switch s {
	case SubjectUser:
		return "user"
	case SubjectAPIKey:
		return "apikey"
	case SubjectVero:
		return "vero"
	default:
		return "unknown"
	}
}
