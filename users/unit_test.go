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
