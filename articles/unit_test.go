package articles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/likui628/golang-gin-realworld-example-app/common"
	"github.com/likui628/golang-gin-realworld-example-app/users"
	"gorm.io/gorm"
)

func newTestArticleService() ArticleService {
	return NewArticleService(NewArticleRepository(common.DB))
}

func newTestArticleHandler() ArticleHandler {
	return NewArticleHandler(newTestArticleService())
}

func setupArticleTestDB(t *testing.T) {
	t.Helper()
	t.Setenv(common.JWTSecretEnvVar, "test-jwt-secret")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	common.DB = db
	users.AutoMigrate(db)
	AutoMigrate(db)
}

func seedArticleAuthor(t *testing.T) users.UserModel {
	t.Helper()

	service := users.NewUserService(users.NewUserRepository(common.DB))
	registered, err := service.Register(users.RegisterUserInput{
		Username: "articleuser",
		Email:    "article@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to seed article author: %v", err)
	}

	return registered.UserModel
}

func performCreateArticleRequest(t *testing.T, authorizationHeader, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	userService := users.NewUserService(users.NewUserRepository(common.DB))
	articlesGroup := r.Group("/articles")
	articlesGroup.Use(users.AuthMiddleware(userService))
	ArticlesRegister(articlesGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodPost, "/articles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func TestCreateArticleSuccess(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	body := `{"article":{"title":"My First Article","description":"Short summary","body":"Full body text","tagList":["go","gin"]}}`
	resp := performCreateArticleRequest(t, "Token "+common.GenToken(author.ID), body)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	articlePayload, ok := payload["article"]
	if !ok {
		t.Fatalf("expected article payload, got: %s", resp.Body.String())
	}

	if articlePayload["title"] != "My First Article" {
		t.Fatalf("expected title My First Article, got %v", articlePayload["title"])
	}

	if articlePayload["slug"] != "my-first-article" {
		t.Fatalf("expected slug my-first-article, got %v", articlePayload["slug"])
	}

	var saved ArticleModel
	if err := common.DB.Preload("Tags").Where("slug = ?", "my-first-article").First(&saved).Error; err != nil {
		t.Fatalf("expected article persisted in db, query error: %v", err)
	}

	if saved.AuthorId != author.ID {
		t.Fatalf("expected author id %d, got %d", author.ID, saved.AuthorId)
	}

	if saved.Description != "Short summary" {
		t.Fatalf("expected description persisted, got %q", saved.Description)
	}

	if saved.Body != "Full body text" {
		t.Fatalf("expected body persisted, got %q", saved.Body)
	}
	if len(saved.Tags) != 2 {
		t.Fatalf("expected 2 tags persisted, got %d", len(saved.Tags))
	}
	tagSet := map[string]bool{}
	for _, tag := range saved.Tags {
		tagSet[tag.Tag] = true
	}
	if !tagSet["go"] || !tagSet["gin"] {
		t.Fatalf("expected tags 'go' and 'gin', got %v", saved.Tags)
	}
}

func TestCreateArticleValidationError(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	body := `{"article":{"description":"Short summary","body":"Full body text"}}`
	resp := performCreateArticleRequest(t, "Token "+common.GenToken(author.ID), body)

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

	if _, exists := errorsPayload["Title"]; !exists {
		t.Fatalf("expected Title validation error, got: %v", errorsPayload)
	}
}

func TestCreateArticleUnauthorized(t *testing.T) {
	setupArticleTestDB(t)

	body := `{"article":{"title":"My First Article","description":"Short summary","body":"Full body text"}}`
	resp := performCreateArticleRequest(t, "", body)

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

	if errorsPayload["auth"] != users.ErrUnauthorized.Error() {
		t.Fatalf("expected auth error %q, got %v", users.ErrUnauthorized.Error(), errorsPayload["auth"])
	}
}
