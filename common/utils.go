package common

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gosimple/slug"
)

const RandomPassword = "A String Very Very Very Random!!@##$!@#4" // #nosec G101

const JWTSecretEnvVar = "JWT_SECRET"

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrMissingJWTSecret = errors.New("missing jwt secret")
)

func GetJWTSecret() (string, error) {
	jwtSecret := os.Getenv(JWTSecretEnvVar)
	if jwtSecret == "" {
		return "", ErrMissingJWTSecret
	}
	return jwtSecret, nil
}

func GenToken(id uint) string {
	jwtSecret, err := GetJWTSecret()
	if err != nil {
		fmt.Printf("failed to load JWT secret for id %d: %v\n", id, err)
		return ""
	}

	jwt_token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  id,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	// Sign and get the complete encoded token as a string
	token, err := jwt_token.SignedString([]byte(jwtSecret))
	if err != nil {
		fmt.Printf("failed to sign JWT token for id %d: %v\n", id, err)
		return ""
	}
	return token
}

func ParseToken(tokenString string) (uint, error) {
	jwtSecret, err := GetJWTSecret()
	if err != nil {
		return 0, ErrInvalidToken
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return 0, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, ErrInvalidToken
	}

	idValue, ok := claims["id"]
	if !ok {
		return 0, ErrInvalidToken
	}

	switch value := idValue.(type) {
	case float64:
		return uint(value), nil
	case int:
		return uint(value), nil
	case uint:
		return value, nil
	default:
		return 0, ErrInvalidToken
	}
}

func GenerateSlug(title string) string {
	return slug.Make(title)
}

type CommonError struct {
	Errors map[string]interface{} `json:"errors"`
}

func NewValidatorError(err error) CommonError {
	res := CommonError{}
	res.Errors = make(map[string]interface{})
	errs := err.(validator.ValidationErrors)
	for _, v := range errs {
		// can translate each error one at a time.
		//fmt.Println("gg",v.NameNamespace)
		if v.Param() != "" {
			res.Errors[v.Field()] = fmt.Sprintf("{%v: %v}", v.Tag(), v.Param())
		} else {
			res.Errors[v.Field()] = fmt.Sprintf("{key: %v}", v.Tag())
		}

	}
	return res
}

func NewError(key string, err error) CommonError {
	res := CommonError{}
	res.Errors = make(map[string]interface{})
	res.Errors[key] = err.Error()
	return res
}

func Bind(c *gin.Context, obj interface{}) error {
	b := binding.Default(c.Request.Method, c.ContentType())
	return c.ShouldBindWith(obj, b)
}
