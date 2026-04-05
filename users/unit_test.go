package users

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/likui628/golang-gin-realworld-example-app/common"
	"gorm.io/gorm"
)

var image_url = "https://golang.org/doc/gopher/frontpage.png"
var test_db *gorm.DB

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
	UsersRegister(usersGroup)

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
	authorized.Use(AuthMiddleware())
	authorized.GET("", func(c *gin.Context) {
		currentUser, ok := CurrentUser(c)
		if !ok {
			c.JSON(http.StatusInternalServerError, common.NewError("auth", ErrUnauthorized))
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": UserSerializer{User: currentUser}.Response()})
	})

	req := httptest.NewRequest(http.MethodGet, "/user", nil)
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
	UsersRegister(usersGroup)

	req := httptest.NewRequest(http.MethodPost, "/users/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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
	if err := seed.setPassword(password); err != nil {
		t.Fatalf("failed to hash password for seed user: %v", err)
	}
	if err := NewUserRepository(common.DB).Create(&seed); err != nil {
		t.Fatalf("failed to seed user: %v", err)
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
