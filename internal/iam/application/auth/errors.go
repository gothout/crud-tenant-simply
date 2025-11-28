package auth

import "errors"

var (
	ErrPwdWrong        = errors.New("error when logging in")
	ErrTokenDuplicated = errors.New("error when logging in")
	OTPCodeExist       = errors.New("otp code has exist")
	OTPCodeWrong       = errors.New("otp code wrong")
)
