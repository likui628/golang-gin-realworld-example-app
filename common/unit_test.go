package common

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
)

type testBindPayload struct {
	User struct {
		Email string `json:"email" binding:"required,email"`
	} `json:"user"`
}

func TestGetJWTSecret(t *testing.T) {
	t.Setenv(JWTSecretEnvVar, "")

	_, err := GetJWTSecret()
	if !errors.Is(err, ErrMissingJWTSecret) {
		t.Fatalf("expected ErrMissingJWTSecret, got %v", err)
	}

	t.Setenv(JWTSecretEnvVar, "test-secret")

	secret, err := GetJWTSecret()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if secret != "test-secret" {
		t.Fatalf("expected test-secret, got %q", secret)
	}
}

func TestGenTokenAndParseToken(t *testing.T) {
	t.Setenv(JWTSecretEnvVar, "test-secret")

	token := GenToken(42)
	if token == "" {
		t.Fatal("expected a signed token")
	}

	userID, err := ParseToken(token)
	if err != nil {
		t.Fatalf("expected no parse error, got %v", err)
	}
	if userID != 42 {
		t.Fatalf("expected user id 42, got %d", userID)
	}
}

func TestGenTokenWithoutSecretReturnsEmptyString(t *testing.T) {
	t.Setenv(JWTSecretEnvVar, "")

	token := GenToken(42)
	if token != "" {
		t.Fatalf("expected empty token when secret is missing, got %q", token)
	}
}

func TestParseTokenRejectsInvalidInputs(t *testing.T) {
	t.Setenv(JWTSecretEnvVar, "test-secret")

	invalidCases := map[string]string{
		"malformed token": "not-a-token",
	}

	for name, tokenString := range invalidCases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseToken(tokenString)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("expected ErrInvalidToken, got %v", err)
			}
		})
	}

	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"id": 99})
	noneTokenString, err := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to sign none token: %v", err)
	}
	if _, err := ParseToken(noneTokenString); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for disallowed signing method, got %v", err)
	}

	missingIDToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": 42})
	missingIDTokenString, err := missingIDToken.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign token without id: %v", err)
	}
	if _, err := ParseToken(missingIDTokenString); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for missing id claim, got %v", err)
	}

	t.Setenv(JWTSecretEnvVar, "")
	if _, err := ParseToken(missingIDTokenString); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken when secret is missing, got %v", err)
	}
}

func TestGenerateSlug(t *testing.T) {
	slug := GenerateSlug("Hello, Real World in Go!")
	if slug != "hello-real-world-in-go" {
		t.Fatalf("expected slug hello-real-world-in-go, got %q", slug)
	}
}

func TestNewValidatorError(t *testing.T) {
	type sample struct {
		Email string `validate:"required,email"`
		Name  string `validate:"required"`
	}

	validate := validator.New()
	err := validate.Struct(sample{})
	if err == nil {
		t.Fatal("expected validation error")
	}

	result := NewValidatorError(err)
	if !strings.Contains(result.Errors["Email"].(string), "required") {
		t.Fatalf("expected Email error to mention required, got %v", result.Errors["Email"])
	}
	if result.Errors["Name"] != "{key: required}" {
		t.Fatalf("expected Name required error, got %v", result.Errors["Name"])
	}
}

func TestNewError(t *testing.T) {
	err := NewError("auth", errors.New("unauthorized"))
	if err.Errors["auth"] != "unauthorized" {
		t.Fatalf("expected unauthorized message, got %v", err.Errors["auth"])
	}
}

func TestBindJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"user":{"email":"user@example.com"}}`))
	request.Header.Set("Content-Type", "application/json")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	var payload testBindPayload
	if err := Bind(context, &payload); err != nil {
		t.Fatalf("expected bind to succeed, got %v", err)
	}
	if payload.User.Email != "user@example.com" {
		t.Fatalf("expected parsed email, got %q", payload.User.Email)
	}
}

func TestGetDBPath(t *testing.T) {
	t.Setenv("DB_PATH", "")
	if path := GetDBPath(); path != "./data/gorm.db" {
		t.Fatalf("expected default DB path, got %q", path)
	}

	t.Setenv("DB_PATH", "./data/test.db")
	if path := GetDBPath(); path != "./data/test.db" {
		t.Fatalf("expected env DB path, got %q", path)
	}
}

func TestInitDatabaseSetsGlobalDB(t *testing.T) {
	dbFile := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("DB_PATH", dbFile)

	db := InitDatabase()
	if db == nil {
		t.Fatal("expected database connection")
	}
	if GetDB() != db {
		t.Fatal("expected InitDatabase to update global DB")
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("expected sql.DB handle, got %v", err)
	}
	t.Cleanup(func() {
		sqlDB.Close()
	})

	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("expected sqlite file to exist, got %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("expected database ping to succeed, got %v", err)
	}
}
