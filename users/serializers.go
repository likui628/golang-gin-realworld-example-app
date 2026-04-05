package users

type UserSerializer struct {
	User UserOutput
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
	return UserResponse{
		Username: serializer.User.Username,
		Email:    serializer.User.Email,
		Bio:      serializer.User.Bio,
		Image:    image,
		Token:    serializer.User.Token,
	}
}
