package middlewares

import (
	"fmt"
	"go-ws/utils/errs"
	"go-ws/utils/logger"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

)

const (
	secret = "s7C28QWAu@l%n-4Mb!9$w@%7fc+0*m%wP&7mAKJao52aU@cGw8IexbNdIF5G!Knk"
)

// LoginInfo struct
type LoginInfo struct {
	UID   int32
	UserName string
}

// LoginAuth middleware
func LoginAuth() gin.HandlerFunc {
	return func(c *gin.Context) {

		token := c.Request.Header.Get("token")

		if token == "" {
			token = c.Query("token")
		}
		if len(token) == 0 {
			logger.Logger.Warn("login auth check is null", zap.String("token", token))
			c.AbortWithError(http.StatusOK, c.Error(errs.ErrUnLogin))
			return
		}

		var str string
		fmt.Sscanf(token, "%s", &str)

		loginInfo, err := parseJwt(str, secret)

		if err != nil {
			logger.Logger.Warn("login auth check error", zap.String("token", token), zap.Error(err))
			c.AbortWithError(http.StatusOK, c.Error(errs.ErrUnLogin))
			return
		}
		if loginInfo.UID == 0 {
			logger.Logger.Warn("login auth check error", zap.String("token", token), zap.Error(err))
			c.AbortWithError(http.StatusOK, c.Error(errs.ErrUnLogin))
			return
		}

		c.Set("uid", loginInfo.UID)
		c.Set("username", loginInfo.UserName)
		c.Next()
	}
}

// secretFunc validates the secret format.
func secretFunc(secret string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		// Make sure the `alg` is what we except.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.Logger.Error("token.Method.(*jwt.SigningMethodHMACe error")
			return nil, jwt.ErrSignatureInvalid
		}

		return []byte(secret), nil
	}
}

func parseJwt(tokenString string, secret string) (*LoginInfo, error) {
	loginInfo := &LoginInfo{}
	token, err := jwt.Parse(tokenString, secretFunc(secret))
	if err != nil {
		logger.Logger.Error("jwt.Parse error", zap.String("tokenString", tokenString), zap.String("secret", secret), zap.Error(err))
		return loginInfo, err
	}
	//check token.Valid
	if !token.Valid {
		logger.Logger.Error("!token.Valid", zap.String("tokenString", tokenString), zap.String("secret", secret), zap.Error(err))
		return loginInfo, errs.ErrUnLogin
	}
	claims, bOK := token.Claims.(jwt.MapClaims)
	if !bOK {
		logger.Logger.Error("get Claims error", zap.String("tokenString", tokenString), zap.String("secret", secret), zap.Error(err))
		return loginInfo, errs.ErrUnLogin
	}
	loginInfo.UID = int32(claims["mid"].(float64))
	loginInfo.UserName = claims["name"].(string)
	return loginInfo, nil
}

// Sign method
func Sign() (tokenString string, err error) {
	// The token content.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":  "test",
		"exp":  1599868536,
		"sub":  "huoren.com",
		"aud":  "all",
		"nbf":  1570526741,
		"iat":  1570526741,
		"jti":  13855,
		"mid":  1000,
		"name": "ws",
	})
	tokenString, err = token.SignedString([]byte(secret))
	return
}
