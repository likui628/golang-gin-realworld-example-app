package articles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	requireRealWorldArticleResponse(t, articlePayload, "articleuser")

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

func performGetArticleRequest(t *testing.T, slug, authorizationHeader string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	userService := users.NewUserService(users.NewUserRepository(common.DB))
	articlesGroup := r.Group("/articles")
	articlesGroup.Use(users.AuthMiddleware(userService))
	ArticlesRegister(articlesGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodGet, "/articles/"+slug, nil)
	req.Header.Set("Content-Type", "application/json")
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performFavoriteArticleRequest(t *testing.T, slug, authorizationHeader string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	userService := users.NewUserService(users.NewUserRepository(common.DB))
	articlesGroup := r.Group("/articles")
	articlesGroup.Use(users.AuthMiddleware(userService))
	ArticlesRegister(articlesGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodPost, "/articles/"+slug+"/favorite", nil)
	req.Header.Set("Content-Type", "application/json")
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performUnfavoriteArticleRequest(t *testing.T, slug, authorizationHeader string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	userService := users.NewUserService(users.NewUserRepository(common.DB))
	articlesGroup := r.Group("/articles")
	articlesGroup.Use(users.AuthMiddleware(userService))
	ArticlesRegister(articlesGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodDelete, "/articles/"+slug+"/favorite", nil)
	req.Header.Set("Content-Type", "application/json")
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func createArticleAndReturnSlug(t *testing.T, authorID uint, body string) string {
	t.Helper()

	resp := performCreateArticleRequest(t, "Token "+common.GenToken(authorID), body)
	if resp.Code != http.StatusCreated {
		t.Fatalf("failed to create article: status %d, body: %s", resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse create response: %v", err)
	}

	articlePayload, ok := payload["article"]
	if !ok {
		t.Fatalf("expected article in create response: %s", resp.Body.String())
	}

	slug, ok := articlePayload["slug"].(string)
	if !ok {
		t.Fatalf("expected slug string in create response, got: %v", articlePayload["slug"])
	}

	return slug
}

func requireRealWorldStringField(t *testing.T, payload map[string]interface{}, field string) string {
	t.Helper()

	value, ok := payload[field]
	if !ok {
		t.Fatalf("expected %q field, got %v", field, payload)
	}

	text, ok := value.(string)
	if !ok || text == "" {
		t.Fatalf("expected non-empty string field %q, got %T(%v)", field, value, value)
	}

	return text
}

func requireRealWorldISOTimeField(t *testing.T, payload map[string]interface{}, field string) {
	t.Helper()

	value := requireRealWorldStringField(t, payload, field)
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		t.Fatalf("expected %q to be RFC3339 timestamp, got %q: %v", field, value, err)
	}
}

func requireRealWorldBoolField(t *testing.T, payload map[string]interface{}, field string) bool {
	t.Helper()

	value, ok := payload[field]
	if !ok {
		t.Fatalf("expected %q field, got %v", field, payload)
	}

	booleanValue, ok := value.(bool)
	if !ok {
		t.Fatalf("expected bool field %q, got %T(%v)", field, value, value)
	}

	return booleanValue
}

func requireRealWorldIntegerField(t *testing.T, payload map[string]interface{}, field string) int64 {
	t.Helper()

	value, ok := payload[field]
	if !ok {
		t.Fatalf("expected %q field, got %v", field, payload)
	}

	numberValue, ok := value.(float64)
	if !ok {
		t.Fatalf("expected numeric field %q, got %T(%v)", field, value, value)
	}

	integerValue := int64(numberValue)
	if float64(integerValue) != numberValue {
		t.Fatalf("expected integer field %q, got %v", field, numberValue)
	}

	return integerValue
}

func requireRealWorldStringArrayField(t *testing.T, payload map[string]interface{}, field string) []string {
	t.Helper()

	value, ok := payload[field]
	if !ok {
		t.Fatalf("expected %q field, got %v", field, payload)
	}

	rawItems, ok := value.([]interface{})
	if !ok {
		t.Fatalf("expected array field %q, got %T(%v)", field, value, value)
	}

	items := make([]string, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, ok := rawItem.(string)
		if !ok {
			t.Fatalf("expected %q items to be strings, got %T(%v)", field, rawItem, rawItem)
		}
		items = append(items, item)
	}

	return items
}

func requireRealWorldAuthorField(t *testing.T, payload map[string]interface{}, expectedUsername string) {
	t.Helper()

	value, ok := payload["author"]
	if !ok {
		t.Fatalf("expected \"author\" field, got %v", payload)
	}

	authorPayload, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected author to be an object, got %T(%v)", value, value)
	}

	username := requireRealWorldStringField(t, authorPayload, "username")
	if expectedUsername != "" && username != expectedUsername {
		t.Fatalf("expected author.username %q, got %q", expectedUsername, username)
	}

	if _, ok := authorPayload["bio"]; !ok {
		t.Fatalf("expected author.bio field, got %v", authorPayload)
	}

	if _, ok := authorPayload["image"]; !ok {
		t.Fatalf("expected author.image field, got %v", authorPayload)
	}

	requireRealWorldBoolField(t, authorPayload, "following")
}

func requireRealWorldArticleResponse(t *testing.T, payload map[string]interface{}, expectedAuthor string) {
	t.Helper()

	requireRealWorldStringField(t, payload, "title")
	requireRealWorldStringField(t, payload, "slug")
	requireRealWorldStringField(t, payload, "description")
	requireRealWorldStringField(t, payload, "body")
	requireRealWorldISOTimeField(t, payload, "createdAt")
	requireRealWorldISOTimeField(t, payload, "updatedAt")
	requireRealWorldStringArrayField(t, payload, "tagList")
	requireRealWorldBoolField(t, payload, "favorited")
	requireRealWorldIntegerField(t, payload, "favoritesCount")
	requireRealWorldAuthorField(t, payload, expectedAuthor)
}

func TestGetArticleBySlugSuccess(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article with tags
	createBody := `{"article":{"title":"Test Article","description":"Test description","body":"Test body","tagList":["golang","testing"]}}`
	createResp := performCreateArticleRequest(t, "Token "+common.GenToken(author.ID), createBody)

	if createResp.Code != http.StatusCreated {
		t.Fatalf("failed to create article: status %d, body: %s", createResp.Code, createResp.Body.String())
	}

	var createPayload map[string]map[string]interface{}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("failed to parse create response: %v", err)
	}

	articleData, ok := createPayload["article"]
	if !ok {
		t.Fatalf("expected article in create response: %s", createResp.Body.String())
	}

	slug, ok := articleData["slug"].(string)
	if !ok {
		t.Fatalf("expected slug in article response, got: %v", articleData["slug"])
	}

	// Get the article by slug
	getResp := performGetArticleRequest(t, slug, "Token "+common.GenToken(author.ID))

	if getResp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, getResp.Code, getResp.Body.String())
	}

	var getPayload map[string]map[string]interface{}
	if err := json.Unmarshal(getResp.Body.Bytes(), &getPayload); err != nil {
		t.Fatalf("failed to parse get response: %v", err)
	}

	retrievedArticle, ok := getPayload["article"]
	if !ok {
		t.Fatalf("expected article in get response: %s", getResp.Body.String())
	}
	requireRealWorldArticleResponse(t, retrievedArticle, "articleuser")

	if retrievedArticle["title"] != "Test Article" {
		t.Fatalf("expected title 'Test Article', got %v", retrievedArticle["title"])
	}

	if retrievedArticle["slug"] != slug {
		t.Fatalf("expected slug %q, got %v", slug, retrievedArticle["slug"])
	}

	if retrievedArticle["description"] != "Test description" {
		t.Fatalf("expected description 'Test description', got %v", retrievedArticle["description"])
	}

	if retrievedArticle["body"] != "Test body" {
		t.Fatalf("expected body 'Test body', got %v", retrievedArticle["body"])
	}

	// Verify tags are returned
	tagList, ok := retrievedArticle["tagList"].([]interface{})
	if !ok {
		t.Fatalf("expected tagList array, got: %v", retrievedArticle["tagList"])
	}

	if len(tagList) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(tagList), tagList)
	}

	tagSet := map[string]bool{}
	for _, tag := range tagList {
		tagSet[tag.(string)] = true
	}

	if !tagSet["golang"] || !tagSet["testing"] {
		t.Fatalf("expected tags 'golang' and 'testing', got %v", tagList)
	}
}

func TestGetArticleBySlugNotFound(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	resp := performGetArticleRequest(t, "missing-article", "Token "+common.GenToken(author.ID))

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object, got: %s", resp.Body.String())
	}

	if errorsPayload["article"] != gorm.ErrRecordNotFound.Error() {
		t.Fatalf("expected article error %q, got %v", gorm.ErrRecordNotFound.Error(), errorsPayload["article"])
	}
}

func TestFavoriteArticleSuccess(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Favorite Me","description":"desc","body":"body","tagList":["go"]}}`)
	resp := performFavoriteArticleRequest(t, slug, "Token "+common.GenToken(author.ID))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	articlePayload, ok := payload["article"]
	if !ok {
		t.Fatalf("expected article payload, got: %s", resp.Body.String())
	}
	requireRealWorldArticleResponse(t, articlePayload, "articleuser")

	if articlePayload["slug"] != slug {
		t.Fatalf("expected slug %q, got %v", slug, articlePayload["slug"])
	}

	if articlePayload["favorited"] != true {
		t.Fatalf("expected favorited=true, got %v", articlePayload["favorited"])
	}

	favoritesCount, ok := articlePayload["favoritesCount"].(float64)
	if !ok {
		t.Fatalf("expected numeric favoritesCount, got %v", articlePayload["favoritesCount"])
	}

	if favoritesCount != 1 {
		t.Fatalf("expected favoritesCount 1, got %v", favoritesCount)
	}

	var count int64
	if err := common.DB.Model(&FavoriteModel{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count favorites: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 favorite row, got %d", count)
	}
}

func TestFavoriteArticleIsIdempotent(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Favorite Once","description":"desc","body":"body"}}`)
	firstResp := performFavoriteArticleRequest(t, slug, "Token "+common.GenToken(author.ID))
	if firstResp.Code != http.StatusOK {
		t.Fatalf("expected first status %d, got %d, body: %s", http.StatusOK, firstResp.Code, firstResp.Body.String())
	}

	secondResp := performFavoriteArticleRequest(t, slug, "Token "+common.GenToken(author.ID))
	if secondResp.Code != http.StatusOK {
		t.Fatalf("expected second status %d, got %d, body: %s", http.StatusOK, secondResp.Code, secondResp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(secondResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	articlePayload, ok := payload["article"]
	if !ok {
		t.Fatalf("expected article payload, got: %s", secondResp.Body.String())
	}
	requireRealWorldArticleResponse(t, articlePayload, "articleuser")

	favoritesCount, ok := articlePayload["favoritesCount"].(float64)
	if !ok {
		t.Fatalf("expected numeric favoritesCount, got %v", articlePayload["favoritesCount"])
	}

	if favoritesCount != 1 {
		t.Fatalf("expected favoritesCount to remain 1, got %v", favoritesCount)
	}

	var count int64
	if err := common.DB.Model(&FavoriteModel{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count favorites: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 favorite row after duplicate favorite, got %d", count)
	}
}

func TestFavoriteArticleUnauthorized(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Unauthorized Favorite","description":"desc","body":"body"}}`)
	resp := performFavoriteArticleRequest(t, slug, "")

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

func TestUnfavoriteArticleSuccess(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Unfavorite Me","description":"desc","body":"body","tagList":["go"]}}`)
	token := "Token " + common.GenToken(author.ID)

	// First, favorite the article
	favResp := performFavoriteArticleRequest(t, slug, token)
	if favResp.Code != http.StatusOK {
		t.Fatalf("expected favorite status %d, got %d, body: %s", http.StatusOK, favResp.Code, favResp.Body.String())
	}

	// Then unfavorite it
	resp := performUnfavoriteArticleRequest(t, slug, token)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	articlePayload, ok := payload["article"]
	if !ok {
		t.Fatalf("expected article payload, got: %s", resp.Body.String())
	}
	requireRealWorldArticleResponse(t, articlePayload, "articleuser")

	if articlePayload["slug"] != slug {
		t.Fatalf("expected slug %q, got %v", slug, articlePayload["slug"])
	}

	if articlePayload["favorited"] != false {
		t.Fatalf("expected favorited=false, got %v", articlePayload["favorited"])
	}

	favoritesCount, ok := articlePayload["favoritesCount"].(float64)
	if !ok {
		t.Fatalf("expected numeric favoritesCount, got %v", articlePayload["favoritesCount"])
	}

	if favoritesCount != 0 {
		t.Fatalf("expected favoritesCount 0, got %v", favoritesCount)
	}

	var count int64
	if err := common.DB.Model(&FavoriteModel{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count favorites: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 favorite rows, got %d", count)
	}
}

func TestUnfavoriteArticleIsIdempotent(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Unfavorite Once","description":"desc","body":"body"}}`)
	token := "Token " + common.GenToken(author.ID)

	// Unfavorite without ever favoriting (should be idempotent)
	firstResp := performUnfavoriteArticleRequest(t, slug, token)
	if firstResp.Code != http.StatusOK {
		t.Fatalf("expected first status %d, got %d, body: %s", http.StatusOK, firstResp.Code, firstResp.Body.String())
	}

	secondResp := performUnfavoriteArticleRequest(t, slug, token)
	if secondResp.Code != http.StatusOK {
		t.Fatalf("expected second status %d, got %d, body: %s", http.StatusOK, secondResp.Code, secondResp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(secondResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	articlePayload, ok := payload["article"]
	if !ok {
		t.Fatalf("expected article payload, got: %s", secondResp.Body.String())
	}
	requireRealWorldArticleResponse(t, articlePayload, "articleuser")

	if articlePayload["favorited"] != false {
		t.Fatalf("expected favorited=false, got %v", articlePayload["favorited"])
	}

	favoritesCount, ok := articlePayload["favoritesCount"].(float64)
	if !ok {
		t.Fatalf("expected numeric favoritesCount, got %v", articlePayload["favoritesCount"])
	}

	if favoritesCount != 0 {
		t.Fatalf("expected favoritesCount to remain 0, got %v", favoritesCount)
	}

	var count int64
	if err := common.DB.Model(&FavoriteModel{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count favorites: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 favorite rows after duplicate unfavorite, got %d", count)
	}
}

func TestUnfavoriteArticleUnauthorized(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Unauthorized Unfavorite","description":"desc","body":"body"}}`)
	resp := performUnfavoriteArticleRequest(t, slug, "")

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

func performGetTagsRequest(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	tagsGroup := r.Group("/tags")
	TagsRegister(tagsGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodGet, "/tags", nil)
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func TestGetTagsEmpty(t *testing.T) {
	setupArticleTestDB(t)

	resp := performGetTagsRequest(t)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	tagsPayload, ok := payload["tags"]
	if !ok {
		t.Fatalf("expected tags field in response, got: %s", resp.Body.String())
	}

	tags, ok := tagsPayload.([]interface{})
	if !ok {
		t.Fatalf("expected tags to be array, got: %v", tagsPayload)
	}

	if len(tags) != 0 {
		t.Fatalf("expected empty tags list, got %d tags", len(tags))
	}
}

func TestGetTagsSuccess(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create multiple articles with different tags
	createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Article 1","description":"desc","body":"body","tagList":["golang","testing"]}}`)
	createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Article 2","description":"desc","body":"body","tagList":["golang","docker"]}}`)
	createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Article 3","description":"desc","body":"body","tagList":["kubernetes"]}}`)

	resp := performGetTagsRequest(t)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	tagsPayload, ok := payload["tags"]
	if !ok {
		t.Fatalf("expected tags field in response, got: %s", resp.Body.String())
	}

	tags, ok := tagsPayload.([]interface{})
	if !ok {
		t.Fatalf("expected tags to be array, got: %v", tagsPayload)
	}

	// Should have 4 unique tags: golang, testing, docker, kubernetes
	if len(tags) != 4 {
		t.Fatalf("expected 4 tags, got %d: %v", len(tags), tags)
	}

	// Create a set to check all expected tags are present
	tagSet := map[string]bool{}
	for _, tag := range tags {
		tagStr, ok := tag.(string)
		if !ok {
			t.Fatalf("expected tag to be string, got: %v", tag)
		}
		tagSet[tagStr] = true
	}

	expectedTags := []string{"golang", "testing", "docker", "kubernetes"}
	for _, expectedTag := range expectedTags {
		if !tagSet[expectedTag] {
			t.Fatalf("expected tag %q not found in response, got: %v", expectedTag, tags)
		}
	}
}

func TestGetTagsWithDuplicates(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create articles with overlapping tags
	createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Article A","description":"desc","body":"body","tagList":["go","rust","python"]}}`)
	createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Article B","description":"desc","body":"body","tagList":["go","python","java"]}}`)

	resp := performGetTagsRequest(t)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	tagsPayload, ok := payload["tags"]
	if !ok {
		t.Fatalf("expected tags field in response, got: %s", resp.Body.String())
	}

	tags, ok := tagsPayload.([]interface{})
	if !ok {
		t.Fatalf("expected tags to be array, got: %v", tagsPayload)
	}

	// Should have 4 unique tags: go, rust, python, java (duplicates removed)
	if len(tags) != 4 {
		t.Fatalf("expected 4 unique tags, got %d: %v", len(tags), tags)
	}

	// Verify all expected tags are present
	tagSet := map[string]bool{}
	for _, tag := range tags {
		tagStr, ok := tag.(string)
		if !ok {
			t.Fatalf("expected tag to be string, got: %v", tag)
		}
		tagSet[tagStr] = true
	}

	expectedTags := []string{"go", "rust", "python", "java"}
	for _, expectedTag := range expectedTags {
		if !tagSet[expectedTag] {
			t.Fatalf("expected tag %q not found in response, got: %v", expectedTag, tags)
		}
	}
}

func performCreateCommentRequest(t *testing.T, slug, authorizationHeader, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	userService := users.NewUserService(users.NewUserRepository(common.DB))
	articlesGroup := r.Group("/articles")
	articlesGroup.Use(users.AuthMiddleware(userService))
	ArticlesRegister(articlesGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodPost, "/articles/"+slug+"/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func TestCreateCommentSuccess(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article
	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Comment Me","description":"desc","body":"body"}}`)

	// Create a comment
	body := `{"comment":{"body":"Test comment body"}}`
	resp := performCreateCommentRequest(t, slug, "Token "+common.GenToken(author.ID), body)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	commentPayload, ok := payload["comment"]
	if !ok {
		t.Fatalf("expected comment payload, got: %s", resp.Body.String())
	}

	// Verify ID is an integer
	id, ok := commentPayload["id"].(float64)
	if !ok || id != float64(int(id)) {
		t.Fatalf("expected integer id, got %v", commentPayload["id"])
	}

	// Verify body
	if commentPayload["body"] != "Test comment body" {
		t.Fatalf("expected body 'Test comment body', got %v", commentPayload["body"])
	}

	// Verify timestamps in ISO 8601 format
	createdAt, ok := commentPayload["createdAt"].(string)
	if !ok {
		t.Fatalf("expected string createdAt, got %v", commentPayload["createdAt"])
	}
	if !isValidISO8601(createdAt) {
		t.Fatalf("expected valid ISO 8601 createdAt, got %q", createdAt)
	}

	updatedAt, ok := commentPayload["updatedAt"].(string)
	if !ok {
		t.Fatalf("expected string updatedAt, got %v", commentPayload["updatedAt"])
	}
	if !isValidISO8601(updatedAt) {
		t.Fatalf("expected valid ISO 8601 updatedAt, got %q", updatedAt)
	}

	// Verify author
	author2, ok := commentPayload["author"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected author object, got %v", commentPayload["author"])
	}

	if author2["username"] != "articleuser" {
		t.Fatalf("expected username 'articleuser', got %v", author2["username"])
	}

	// Verify comment persisted in database
	var savedComment CommentModel
	if err := common.DB.Preload("Author").Where("body = ?", "Test comment body").First(&savedComment).Error; err != nil {
		t.Fatalf("expected comment persisted in db, query error: %v", err)
	}

	if savedComment.AuthorId != author.ID {
		t.Fatalf("expected author id %d, got %d", author.ID, savedComment.AuthorId)
	}
}

func TestCreateCommentValidationError(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article
	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"No Comment","description":"desc","body":"body"}}`)

	// Try to create comment without body
	body := `{"comment":{}}`
	resp := performCreateCommentRequest(t, slug, "Token "+common.GenToken(author.ID), body)

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

	if _, exists := errorsPayload["Body"]; !exists {
		t.Fatalf("expected Body validation error, got: %v", errorsPayload)
	}
}

func TestCreateCommentUnauthorized(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article
	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Unauthorized Comment","description":"desc","body":"body"}}`)

	// Try to create comment without authorization
	body := `{"comment":{"body":"Test comment"}}`
	resp := performCreateCommentRequest(t, slug, "", body)

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

func TestCreateCommentArticleNotFound(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Try to create comment on non-existent article
	body := `{"comment":{"body":"Test comment"}}`
	resp := performCreateCommentRequest(t, "non-existent-slug", "Token "+common.GenToken(author.ID), body)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusInternalServerError, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object, got: %s", resp.Body.String())
	}

	if _, exists := errorsPayload["database"]; !exists {
		t.Fatalf("expected database error, got: %v", errorsPayload)
	}
}

func isValidISO8601(timestamp string) bool {
	// Simple validation for ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ
	if len(timestamp) < 19 {
		return false
	}
	// Check basic pattern - YYYY-MM-DDTHH:MM:SS
	if timestamp[4] != '-' || timestamp[7] != '-' || timestamp[10] != 'T' ||
		timestamp[13] != ':' || timestamp[16] != ':' {
		return false
	}
	return true
}
