package users

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/likui628/golang-gin-realworld-example-app/common"
	"gorm.io/gorm"
)

var image_url = "https://golang.org/doc/gopher/frontpage.png"
var test_db *gorm.DB

func newTestUserService() UserService {
	return NewUserService(NewUserRepository(common.DB))
}

func newTestUserHandler() UserHandler {
	return NewUserHandler(newTestUserService())
}

func setupTestDB(t *testing.T) {
	t.Helper()
	t.Setenv(common.JWTSecretEnvVar, "test-jwt-secret")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	common.DB = db
	test_db = db
	AutoMigrate(db)
}

func performUserRegistrationRequest(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	usersGroup := r.Group("/users")
	UsersRegister(usersGroup, newTestUserHandler())

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performAuthenticatedRequest(t *testing.T, authorizationHeader string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	authorized := r.Group("/user")
	service := newTestUserService()
	authorized.Use(AuthMiddleware(service))
	UserRegister(authorized, newTestUserHandler())

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performUpdateUserRequest(t *testing.T, authorizationHeader, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	authorized := r.Group("/user")
	service := newTestUserService()
	authorized.Use(AuthMiddleware(service))
	UserRegister(authorized, newTestUserHandler())

	req := httptest.NewRequest(http.MethodPut, "/user", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performUserLoginRequest(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	usersGroup := r.Group("/users")
	UsersRegister(usersGroup, newTestUserHandler())

	req := httptest.NewRequest(http.MethodPost, "/users/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performGetProfileRequest(t *testing.T, authorizationHeader string, uid uint) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	profiles := r.Group("/profiles")
	service := newTestUserService()
	profiles.Use(OptionalAuthMiddleware(service))
	ProfileRegister(profiles, newTestUserHandler())

	req := httptest.NewRequest(http.MethodGet, "/profiles/"+strconv.FormatUint(uint64(uid), 10), nil)
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func seedUser(t *testing.T, username, email, password string) {
	t.Helper()

	seed := UserModel{
		Username: username,
		Email:    email,
		Bio:      "hello",
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password for seed user: %v", err)
	}
	seed.PasswordHash = passwordHash
	if err := NewUserRepository(common.DB).Create(&seed); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
}

func seedProfileUser(t *testing.T, username, email, password string) {
	t.Helper()

	seed := UserModel{
		Username: username,
		Email:    email,
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password for profile seed user: %v", err)
	}
	seed.PasswordHash = passwordHash
	if err := NewUserRepository(common.DB).Create(&seed); err != nil {
		t.Fatalf("failed to seed profile user: %v", err)
	}
}

func seedFollow(t *testing.T, followerID uint, followedID uint) {
	t.Helper()

	follow := FollowModel{
		FollowerId: followerID,
		FollowedId: followedID,
	}
	if err := common.DB.Create(&follow).Error; err != nil {
		t.Fatalf("failed to seed follow relation: %v", err)
	}
}

func TestGetProfileWithoutAuth(t *testing.T) {
	setupTestDB(t)
	seedProfileUser(t, "celeb_123", "celeb123@example.com", "password123")
	target, err := NewUserRepository(common.DB).FindByEmail("celeb123@example.com")
	if err != nil {
		t.Fatalf("failed to load target profile: %v", err)
	}

	resp := performGetProfileRequest(t, "", target.ID)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	profilePayload, ok := payload["profile"]
	if !ok {
		t.Fatalf("expected profile payload, got: %s", resp.Body.String())
	}

	if profilePayload["username"] != "celeb_123" {
		t.Fatalf("expected username celeb_123, got %v", profilePayload["username"])
	}

	if value, exists := profilePayload["bio"]; !exists || value != nil {
		t.Fatalf("expected bio to be null, got %v", value)
	}

	if value, exists := profilePayload["image"]; !exists || value != nil {
		t.Fatalf("expected image to be null, got %v", value)
	}

	if following, ok := profilePayload["following"].(bool); !ok || following {
		t.Fatalf("expected following to be false, got %v", profilePayload["following"])
	}
}

func TestGetProfileWithAuth(t *testing.T) {
	setupTestDB(t)
	seedUser(t, "viewer_123", "viewer123@example.com", "password123")
	seedProfileUser(t, "celeb_123", "celeb123@example.com", "password123")

	viewer, err := NewUserRepository(common.DB).FindByEmail("viewer123@example.com")
	if err != nil {
		t.Fatalf("failed to load viewer: %v", err)
	}
	target, err := NewUserRepository(common.DB).FindByEmail("celeb123@example.com")
	if err != nil {
		t.Fatalf("failed to load target profile: %v", err)
	}

	resp := performGetProfileRequest(t, "Token "+common.GenToken(viewer.ID), target.ID)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	profilePayload, ok := payload["profile"]
	if !ok {
		t.Fatalf("expected profile payload, got: %s", resp.Body.String())
	}

	if profilePayload["username"] != "celeb_123" {
		t.Fatalf("expected username celeb_123, got %v", profilePayload["username"])
	}

	if value, exists := profilePayload["bio"]; !exists || value != nil {
		t.Fatalf("expected bio to be null, got %v", value)
	}

	if value, exists := profilePayload["image"]; !exists || value != nil {
		t.Fatalf("expected image to be null, got %v", value)
	}

	if following, ok := profilePayload["following"].(bool); !ok || following {
		t.Fatalf("expected following to be false, got %v", profilePayload["following"])
	}
}

func TestGetProfileWithAuthFollowingTrue(t *testing.T) {
	setupTestDB(t)
	seedUser(t, "viewer_123", "viewer123@example.com", "password123")
	seedProfileUser(t, "celeb_123", "celeb123@example.com", "password123")

	repository := NewUserRepository(common.DB)
	viewer, err := repository.FindByEmail("viewer123@example.com")
	if err != nil {
		t.Fatalf("failed to load viewer: %v", err)
	}
	target, err := repository.FindByEmail("celeb123@example.com")
	if err != nil {
		t.Fatalf("failed to load target profile: %v", err)
	}
	seedFollow(t, viewer.ID, target.ID)

	resp := performGetProfileRequest(t, "Token "+common.GenToken(viewer.ID), target.ID)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	profilePayload, ok := payload["profile"]
	if !ok {
		t.Fatalf("expected profile payload, got: %s", resp.Body.String())
	}

	if following, ok := profilePayload["following"].(bool); !ok || !following {
		t.Fatalf("expected following to be true, got %v", profilePayload["following"])
	}
}

func TestUsersRegistrationSuccess(t *testing.T) {
	setupTestDB(t)

	body := `{"user":{"username":"tester123","email":"tester@example.com","password":"password123","bio":"hello","image":"` + image_url + `"}}`
	resp := performUserRegistrationRequest(t, body)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	userPayload, ok := payload["user"]
	if !ok {
		t.Fatalf("response does not include user object: %s", resp.Body.String())
	}

	if userPayload["username"] != "tester123" {
		t.Fatalf("expected username tester123, got %v", userPayload["username"])
	}

	if userPayload["email"] != "tester@example.com" {
		t.Fatalf("expected email tester@example.com, got %v", userPayload["email"])
	}

	if userPayload["token"] == "" {
		t.Fatalf("expected non-empty token, got empty")
	}

	var saved UserModel
	err := common.DB.Where("email = ?", "tester@example.com").First(&saved).Error
	if err != nil {
		t.Fatalf("expected user persisted in db, query error: %v", err)
	}

	if saved.PasswordHash == "" {
		t.Fatalf("expected password hash to be stored")
	}
}

func TestUsersRegistrationValidationError(t *testing.T) {
	setupTestDB(t)

	body := `{"user":{"username":"abc","email":"invalid-email","password":"123"}}`
	resp := performUserRegistrationRequest(t, body)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errors, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object in response, got: %s", resp.Body.String())
	}

	if _, exists := errors["Username"]; !exists {
		t.Fatalf("expected Username validation error, got: %v", errors)
	}

	if _, exists := errors["Email"]; !exists {
		t.Fatalf("expected Email validation error, got: %v", errors)
	}

	if _, exists := errors["Password"]; !exists {
		t.Fatalf("expected Password validation error, got: %v", errors)
	}
}

func TestUsersRegistrationDuplicateEmail(t *testing.T) {
	setupTestDB(t)
	seedUser(t, "tester123", "tester@example.com", "password123")

	body := `{"user":{"username":"tester456","email":"tester@example.com","password":"password123"}}`
	resp := performUserRegistrationRequest(t, body)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object in response, got: %s", resp.Body.String())
	}

	if errorsPayload["email"] != "has already been taken" {
		t.Fatalf("expected duplicate email error, got: %v", errorsPayload["email"])
	}
}

func TestUsersRegistrationWithoutOptionalFields(t *testing.T) {
	setupTestDB(t)

	body := `{"user":{"username":"tester123","email":"tester@example.com","password":"password123"}}`
	resp := performUserRegistrationRequest(t, body)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	userPayload, ok := payload["user"]
	if !ok {
		t.Fatalf("response does not include user object: %s", resp.Body.String())
	}

	if userPayload["bio"] != "" {
		t.Fatalf("expected empty bio, got %v", userPayload["bio"])
	}

	if userPayload["image"] != "" {
		t.Fatalf("expected empty image, got %v", userPayload["image"])
	}
}

func TestUsersLoginSuccess(t *testing.T) {
	setupTestDB(t)

	seedUser(t, "loginuser", "login@example.com", "password123")

	body := `{"user":{"email":"login@example.com","password":"password123"}}`
	resp := performUserLoginRequest(t, body)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	userPayload, ok := payload["user"]
	if !ok {
		t.Fatalf("response does not include user object: %s", resp.Body.String())
	}

	if userPayload["email"] != "login@example.com" {
		t.Fatalf("expected email login@example.com, got %v", userPayload["email"])
	}

	if userPayload["username"] != "loginuser" {
		t.Fatalf("expected username loginuser, got %v", userPayload["username"])
	}

	if userPayload["token"] == "" {
		t.Fatalf("expected non-empty token, got empty")
	}
}

func TestUsersLoginUnregisteredEmail(t *testing.T) {
	setupTestDB(t)

	body := `{"user":{"email":"missing@example.com","password":"password123"}}`
	resp := performUserLoginRequest(t, body)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object in response, got: %s", resp.Body.String())
	}

	if errorsPayload["login"] != "Not Registered email or invalid password" {
		t.Fatalf("expected login error message, got: %v", errorsPayload["login"])
	}
}

func TestUsersLoginInvalidPassword(t *testing.T) {
	setupTestDB(t)

	seedUser(t, "loginuser2", "login2@example.com", "password123")

	body := `{"user":{"email":"login2@example.com","password":"wrongpass123"}}`
	resp := performUserLoginRequest(t, body)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object in response, got: %s", resp.Body.String())
	}

	if errorsPayload["login"] != "Not Registered email or invalid password" {
		t.Fatalf("expected login error message, got: %v", errorsPayload["login"])
	}
}

func TestUsersLoginValidationError(t *testing.T) {
	setupTestDB(t)

	body := `{"user":{"email":"invalid-email","password":"123"}}`
	resp := performUserLoginRequest(t, body)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object in response, got: %s", resp.Body.String())
	}

	if _, exists := errorsPayload["Email"]; !exists {
		t.Fatalf("expected Email validation error, got: %v", errorsPayload)
	}

	if _, exists := errorsPayload["Password"]; !exists {
		t.Fatalf("expected Password validation error, got: %v", errorsPayload)
	}
}

func TestAuthMiddlewareSuccess(t *testing.T) {
	setupTestDB(t)
	seedUser(t, "authuser", "auth@example.com", "password123")

	userModel, err := NewUserRepository(common.DB).FindByEmail("auth@example.com")
	if err != nil {
		t.Fatalf("failed to load seeded user: %v", err)
	}

	resp := performAuthenticatedRequest(t, "Token "+common.GenToken(userModel.ID))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	userPayload, ok := payload["user"]
	if !ok {
		t.Fatalf("expected user payload, got: %s", resp.Body.String())
	}

	if userPayload["email"] != "auth@example.com" {
		t.Fatalf("expected authenticated email, got %v", userPayload["email"])
	}

	if userPayload["username"] != "authuser" {
		t.Fatalf("expected authenticated username, got %v", userPayload["username"])
	}

	if _, exists := userPayload["bio"]; !exists {
		t.Fatalf("expected bio field, got %v", userPayload)
	}

	if _, exists := userPayload["image"]; !exists {
		t.Fatalf("expected image field, got %v", userPayload)
	}

	if userPayload["token"] == "" {
		t.Fatalf("expected non-empty token, got empty")
	}
}

func TestCurrentUserUnauthorized(t *testing.T) {
	setupTestDB(t)

	resp := performAuthenticatedRequest(t, "")

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object, got: %s", resp.Body.String())
	}

	if errorsPayload["auth"] != ErrUnauthorized.Error() {
		t.Fatalf("expected auth error %q, got %v", ErrUnauthorized.Error(), errorsPayload["auth"])
	}
}

func TestUpdateUserSuccess(t *testing.T) {
	setupTestDB(t)
	seedUser(t, "beforeuser", "before@example.com", "password123")

	userModel, err := NewUserRepository(common.DB).FindByEmail("before@example.com")
	if err != nil {
		t.Fatalf("failed to load seeded user: %v", err)
	}

	body := `{"user":{"email":"after@example.com","username":"afteruser","bio":"updated bio","image":"https://example.com/avatar.png"}}`
	resp := performUpdateUserRequest(t, "Token "+common.GenToken(userModel.ID), body)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	userPayload, ok := payload["user"]
	if !ok {
		t.Fatalf("expected user payload, got: %s", resp.Body.String())
	}

	if userPayload["email"] != "after@example.com" {
		t.Fatalf("expected updated email, got %v", userPayload["email"])
	}

	if userPayload["username"] != "afteruser" {
		t.Fatalf("expected updated username, got %v", userPayload["username"])
	}

	if userPayload["bio"] != "updated bio" {
		t.Fatalf("expected updated bio, got %v", userPayload["bio"])
	}

	if userPayload["image"] != "https://example.com/avatar.png" {
		t.Fatalf("expected updated image, got %v", userPayload["image"])
	}

	if userPayload["token"] == "" {
		t.Fatalf("expected non-empty token, got empty")
	}

	updatedUser, err := NewUserRepository(common.DB).FindByID(userModel.ID)
	if err != nil {
		t.Fatalf("failed to reload updated user: %v", err)
	}

	if updatedUser.Email != "after@example.com" {
		t.Fatalf("expected persisted email after@example.com, got %q", updatedUser.Email)
	}

	if updatedUser.Username != "afteruser" {
		t.Fatalf("expected persisted username afteruser, got %q", updatedUser.Username)
	}

	if updatedUser.Bio != "updated bio" {
		t.Fatalf("expected persisted bio updated bio, got %q", updatedUser.Bio)
	}

	if updatedUser.Image == nil || *updatedUser.Image != "https://example.com/avatar.png" {
		t.Fatalf("expected persisted image, got %v", updatedUser.Image)
	}
}

func TestUpdateUserValidationError(t *testing.T) {
	setupTestDB(t)
	seedUser(t, "updateuser", "update@example.com", "password123")

	userModel, err := NewUserRepository(common.DB).FindByEmail("update@example.com")
	if err != nil {
		t.Fatalf("failed to load seeded user: %v", err)
	}

	body := `{"user":{"email":"invalid-email","username":"abc","image":"not-a-url"}}`
	resp := performUpdateUserRequest(t, "Token "+common.GenToken(userModel.ID), body)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnprocessableEntity, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object, got: %s", resp.Body.String())
	}

	if _, exists := errorsPayload["Email"]; !exists {
		t.Fatalf("expected Email validation error, got: %v", errorsPayload)
	}

	if _, exists := errorsPayload["Username"]; !exists {
		t.Fatalf("expected Username validation error, got: %v", errorsPayload)
	}

	if _, exists := errorsPayload["Image"]; !exists {
		t.Fatalf("expected Image validation error, got: %v", errorsPayload)
	}
}

func TestUpdateUserUnauthorized(t *testing.T) {
	setupTestDB(t)

	body := `{"user":{"email":"after@example.com"}}`
	resp := performUpdateUserRequest(t, "", body)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object, got: %s", resp.Body.String())
	}

	if errorsPayload["auth"] != ErrUnauthorized.Error() {
		t.Fatalf("expected auth error %q, got %v", ErrUnauthorized.Error(), errorsPayload["auth"])
	}
}

func TestAuthMiddlewareMissingToken(t *testing.T) {
	setupTestDB(t)

	resp := performAuthenticatedRequest(t, "")

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	setupTestDB(t)

	resp := performAuthenticatedRequest(t, "Token invalid-token")

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}
}

func TestAuthMiddlewareMissingUser(t *testing.T) {
	setupTestDB(t)

	resp := performAuthenticatedRequest(t, "Token "+common.GenToken(999))

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, resp.Code, resp.Body.String())
	}
}
