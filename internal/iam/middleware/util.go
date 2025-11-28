package middleware

import "github.com/gin-gonic/gin"

const UserContextKey = "AuthenticatedUserKey"

func SetAuthenticatedUser(c *gin.Context, userLogin *Login) {
	if userLogin != nil {
		c.Set(UserContextKey, userLogin)
	}
}
func GetAuthenticatedUser(c *gin.Context) (*Login, bool) {
	value, exists := c.Get(UserContextKey)

	if !exists {
		return nil, false
	}
	userLogin, ok := value.(*Login)

	if !ok {
		return nil, false
	}

	return userLogin, true
}
