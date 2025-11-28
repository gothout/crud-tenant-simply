package auth

import (
	"tenant-crud-simply/internal/iam/domain/user"
	"time"
)

type LoginResponse struct {
	User   user.UserResponseDto `json:"user"`
	Token  string               `json:"token"`
	Expire time.Time            `json:"expire"`
}
