package authenticationService

import (
	"context"
)

type AuthenticationServiceImpl struct {
	context context.Context
}

func NewAuthenticationServiceImpl() (service AuthenticationService, err error) {
	return &AuthenticationServiceImpl{
		context: context.Background(),
	}, err
}
