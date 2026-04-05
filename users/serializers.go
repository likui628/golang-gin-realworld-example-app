package users

import (
	"github.com/likui628/golang-gin-realworld-example-app/common"
)

type UserSerializer struct {
	User UserModel
}

type UserResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Bio      string `json:"bio"`
	Image    string `json:"image"`
	Token    string `json:"token"`
}

func (serializer UserSerializer) Response() UserResponse {
	image := ""
	if serializer.User.Image != nil {
		image = *serializer.User.Image
	}
	user := UserResponse{
		Username: serializer.User.Username,
		Email:    serializer.User.Email,
		Bio:      serializer.User.Bio,
		Image:    image,
		Token:    common.GenToken(serializer.User.ID),
	}
	return user
}
