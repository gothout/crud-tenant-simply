package auth

import (
	"tenant-crud-simply/internal/iam/domain/user"
	"time"
)

type LoginResponse struct {
	User          user.UserResponseDto `json:"user"`
	Token         string               `json:"token"`
	SystemTimeUTC time.Time            `json:"system_time_utc"`
	Expire        time.Time            `json:"expire"`
}
