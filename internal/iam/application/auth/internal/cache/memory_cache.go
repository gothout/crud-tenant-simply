package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var OTPCache = cache.New(5*time.Minute, 10*time.Minute)

func SaveOTP(email, code string) {
	OTPCache.Set(email, code, 5*time.Minute)
}

func GetOTP(email string) (string, bool) {
	v, found := OTPCache.Get(email)
	if !found {
		return "", false
	}
	return v.(string), true
}

func DeleteOTP(email string) {
	OTPCache.Delete(email)
}
