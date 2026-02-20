package authenticationService

import (
	"r2-notify-server/data"
)

type AuthenticationService interface {
	GoogleAuthenticate(token string) (user data.UserInfo, jwt string, err error)
}
