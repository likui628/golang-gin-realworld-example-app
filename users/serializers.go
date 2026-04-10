package users

type UserSerializer struct {
	User UserOutput
}

type ProfileSerializer struct {
	Profile UserProfileOutput
}

type UserResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Bio      string `json:"bio"`
	Image    string `json:"image"`
	Token    string `json:"token"`
}

type ProfileResponse struct {
	Username  string  `json:"username"`
	Bio       *string `json:"bio"`
	Image     *string `json:"image"`
	Following bool    `json:"following"`
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

func (serializer ProfileSerializer) Response() ProfileResponse {
	var bio *string
	if serializer.Profile.Bio != "" {
		bioValue := serializer.Profile.Bio
		bio = &bioValue
	}

	return ProfileResponse{
		Username:  serializer.Profile.Username,
		Bio:       bio,
		Image:     serializer.Profile.Image,
		Following: serializer.Profile.Following,
	}
}
