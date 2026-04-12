package articles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	return NewArticleService(NewArticleRepository(common.DB), users.NewUserRepository(common.DB))
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

	return seedArticleUser(t, "articleuser", "article@example.com")
}

func seedArticleUser(t *testing.T, username, email string) users.UserModel {
	t.Helper()

	service := users.NewUserService(users.NewUserRepository(common.DB))
	registered, err := service.Register(users.RegisterUserInput{
		Username: username,
		Email:    email,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to seed article user: %v", err)
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

func performDeleteArticleRequest(t *testing.T, slug, authorizationHeader string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	userService := users.NewUserService(users.NewUserRepository(common.DB))
	articlesGroup := r.Group("/articles")
	articlesGroup.Use(users.AuthMiddleware(userService))
	ArticlesRegister(articlesGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodDelete, "/articles/"+slug, nil)
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
	articlesGroup.Use(users.OptionalAuthMiddleware(userService))
	ArticlePublicRegister(articlesGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodGet, "/articles/"+slug, nil)
	req.Header.Set("Content-Type", "application/json")
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performGetArticlesRequest(t *testing.T, rawQuery, authorizationHeader string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	userService := users.NewUserService(users.NewUserRepository(common.DB))
	articlesGroup := r.Group("/articles")
	articlesGroup.Use(users.OptionalAuthMiddleware(userService))
	ArticlePublicRegister(articlesGroup, newTestArticleHandler())

	path := "/articles"
	if rawQuery != "" {
		path += "?" + rawQuery
	}

	req := httptest.NewRequest(http.MethodGet, path, nil)
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

	authorPayload, ok := retrievedArticle["author"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected author object, got %v", retrievedArticle["author"])
	}

	if following, ok := authorPayload["following"].(bool); !ok || following {
		t.Fatalf("expected author.following to be false for self lookup, got %v", authorPayload["following"])
	}
}

func TestGetArticleBySlugAuthorFollowingTrue(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)
	viewer := seedArticleUser(t, "articleviewer", "viewer@example.com")

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Followed Author","description":"desc","body":"body","tagList":["go"]}}`)
	if err := users.NewUserRepository(common.DB).FollowUser(viewer.ID, author.ID); err != nil {
		t.Fatalf("failed to follow article author: %v", err)
	}

	resp := performGetArticleRequest(t, slug, "Token "+common.GenToken(viewer.ID))
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

	authorPayload, ok := articlePayload["author"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected author object, got %v", articlePayload["author"])
	}

	if following, ok := authorPayload["following"].(bool); !ok || !following {
		t.Fatalf("expected author.following to be true, got %v", authorPayload["following"])
	}
}

func TestGetArticlesFilterByAuthorUsername(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleUser(t, "author-one", "author-one@example.com")
	otherAuthor := seedArticleUser(t, "author-two", "author-two@example.com")

	createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Author One Article","description":"desc","body":"body","tagList":["go"]}}`)
	createArticleAndReturnSlug(t, otherAuthor.ID, `{"article":{"title":"Author Two Article","description":"desc","body":"body","tagList":["gin"]}}`)

	resp := performGetArticlesRequest(t, "author=author-one", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	articlesPayload, ok := payload["articles"].([]interface{})
	if !ok {
		t.Fatalf("expected articles array, got: %v", payload["articles"])
	}
	if len(articlesPayload) != 1 {
		t.Fatalf("expected 1 article, got %d: %v", len(articlesPayload), articlesPayload)
	}
	if articleCount := int64(payload["articleCount"].(float64)); articleCount != 1 {
		t.Fatalf("expected articleCount 1, got %d", articleCount)
	}

	article, ok := articlesPayload[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected article object, got: %v", articlesPayload[0])
	}
	requireRealWorldArticleResponse(t, article, "author-one")
	if article["title"] != "Author One Article" {
		t.Fatalf("expected filtered article title %q, got %v", "Author One Article", article["title"])
	}
}

func TestGetArticlesFilterByTag(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Tagged Go","description":"desc","body":"body","tagList":["go","backend"]}}`)
	createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Tagged Gin","description":"desc","body":"body","tagList":["gin"]}}`)

	resp := performGetArticlesRequest(t, "tag=backend", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	articlesPayload, ok := payload["articles"].([]interface{})
	if !ok {
		t.Fatalf("expected articles array, got: %v", payload["articles"])
	}
	if len(articlesPayload) != 1 {
		t.Fatalf("expected 1 article, got %d: %v", len(articlesPayload), articlesPayload)
	}
	if articleCount := int64(payload["articleCount"].(float64)); articleCount != 1 {
		t.Fatalf("expected articleCount 1, got %d", articleCount)
	}

	article, ok := articlesPayload[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected article object, got: %v", articlesPayload[0])
	}
	requireRealWorldArticleResponse(t, article, "articleuser")
	if article["title"] != "Tagged Go" {
		t.Fatalf("expected filtered article title %q, got %v", "Tagged Go", article["title"])
	}

	tagList := requireRealWorldStringArrayField(t, article, "tagList")
	tagSet := map[string]bool{}
	for _, tag := range tagList {
		tagSet[tag] = true
	}
	if !tagSet["backend"] {
		t.Fatalf("expected filtered article to include tag %q, got %v", "backend", tagList)
	}
}

func TestGetArticlesInvalidLimit(t *testing.T) {
	setupArticleTestDB(t)

	resp := performGetArticlesRequest(t, "limit=invalid", "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object, got: %s", resp.Body.String())
	}

	if errorsPayload["limit"] != "invalid limit" {
		t.Fatalf("expected limit error %q, got %v", "invalid limit", errorsPayload["limit"])
	}
}

func TestGetArticlesInvalidOffset(t *testing.T) {
	setupArticleTestDB(t)

	resp := performGetArticlesRequest(t, "offset=-1", "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object, got: %s", resp.Body.String())
	}

	if errorsPayload["offset"] != "invalid offset" {
		t.Fatalf("expected offset error %q, got %v", "invalid offset", errorsPayload["offset"])
	}
}

func TestGetArticlesIncludesViewerState(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleUser(t, "author-followed", "author-followed@example.com")
	viewer := seedArticleUser(t, "article-viewer", "article-viewer@example.com")

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Viewer State","description":"desc","body":"body","tagList":["go"]}}`)
	if err := users.NewUserRepository(common.DB).FollowUser(viewer.ID, author.ID); err != nil {
		t.Fatalf("failed to follow article author: %v", err)
	}

	service := newTestArticleService()
	if _, err := service.FavoriteArticle(viewer.ID, slug); err != nil {
		t.Fatalf("failed to favorite article: %v", err)
	}

	resp := performGetArticlesRequest(t, "author=author-followed", "Token "+common.GenToken(viewer.ID))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	articlesPayload, ok := payload["articles"].([]interface{})
	if !ok {
		t.Fatalf("expected articles array, got: %v", payload["articles"])
	}
	if len(articlesPayload) != 1 {
		t.Fatalf("expected 1 article, got %d: %v", len(articlesPayload), articlesPayload)
	}

	articlePayload, ok := articlesPayload[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected article object, got: %v", articlesPayload[0])
	}
	requireRealWorldArticleResponse(t, articlePayload, "author-followed")

	if favorited, ok := articlePayload["favorited"].(bool); !ok || !favorited {
		t.Fatalf("expected favorited=true, got %v", articlePayload["favorited"])
	}

	if favoritesCount, ok := articlePayload["favoritesCount"].(float64); !ok || favoritesCount != 1 {
		t.Fatalf("expected favoritesCount=1, got %v", articlePayload["favoritesCount"])
	}

	authorPayload, ok := articlePayload["author"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected author object, got %v", articlePayload["author"])
	}

	if following, ok := authorPayload["following"].(bool); !ok || !following {
		t.Fatalf("expected author.following=true, got %v", authorPayload["following"])
	}
}

func TestDeleteArticleSuccess(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Delete Me","description":"desc","body":"body","tagList":["go"]}}`)
	resp := performDeleteArticleRequest(t, slug, "Token "+common.GenToken(author.ID))
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusNoContent, resp.Code, resp.Body.String())
	}

	var count int64
	if err := common.DB.Model(&ArticleModel{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
		t.Fatalf("failed to verify article deletion: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected article to be deleted, found %d rows", count)
	}
}

func TestDeleteArticleUnauthorized(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Protected Delete","description":"desc","body":"body"}}`)
	resp := performDeleteArticleRequest(t, slug, "")
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

	var count int64
	if err := common.DB.Model(&ArticleModel{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
		t.Fatalf("failed to verify article remained persisted: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected article to remain after unauthorized delete, found %d rows", count)
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

func performGetCommentsRequest(t *testing.T, slug, authorizationHeader string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	publicArticlesGroup := r.Group("/articles")
	ArticlePublicRegister(publicArticlesGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodGet, "/articles/"+slug+"/comments", nil)
	req.Header.Set("Content-Type", "application/json")
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performDeleteCommentRequest(t *testing.T, slug, commentID, authorizationHeader string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	userService := users.NewUserService(users.NewUserRepository(common.DB))
	articlesGroup := r.Group("/articles")
	articlesGroup.Use(users.AuthMiddleware(userService))
	ArticlesRegister(articlesGroup, newTestArticleHandler())

	req := httptest.NewRequest(http.MethodDelete, "/articles/"+slug+"/comments/"+commentID, nil)
	req.Header.Set("Content-Type", "application/json")
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func TestGetCommentsSuccess(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article
	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Article With Comments","description":"desc","body":"body"}}`)

	// Create multiple comments
	resp1 := performCreateCommentRequest(t, slug, "Token "+common.GenToken(author.ID), `{"comment":{"body":"First comment"}}`)
	if resp1.Code != http.StatusCreated {
		t.Fatalf("failed to create first comment: status %d", resp1.Code)
	}

	resp2 := performCreateCommentRequest(t, slug, "Token "+common.GenToken(author.ID), `{"comment":{"body":"Second comment"}}`)
	if resp2.Code != http.StatusCreated {
		t.Fatalf("failed to create second comment: status %d", resp2.Code)
	}

	resp3 := performCreateCommentRequest(t, slug, "Token "+common.GenToken(author.ID), `{"comment":{"body":"Third comment"}}`)
	if resp3.Code != http.StatusCreated {
		t.Fatalf("failed to create third comment: status %d", resp3.Code)
	}

	// Get all comments
	resp := performGetCommentsRequest(t, slug, "Token "+common.GenToken(author.ID))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	commentsPayload, ok := payload["comments"]
	if !ok {
		t.Fatalf("expected comments field, got: %s", resp.Body.String())
	}

	comments, ok := commentsPayload.([]interface{})
	if !ok {
		t.Fatalf("expected comments to be array, got: %v", commentsPayload)
	}

	if len(comments) != 3 {
		t.Fatalf("expected 3 comments, got %d", len(comments))
	}

	// Verify all comments have required fields
	for i, commentInterface := range comments {
		comment, ok := commentInterface.(map[string]interface{})
		if !ok {
			t.Fatalf("expected comment %d to be object, got %T", i, commentInterface)
		}

		// Verify ID is an integer
		id, ok := comment["id"].(float64)
		if !ok || id != float64(int(id)) {
			t.Fatalf("expected integer id for comment %d, got %v", i, comment["id"])
		}

		// Verify body exists
		_, ok = comment["body"].(string)
		if !ok {
			t.Fatalf("expected string body for comment %d, got %v", i, comment["body"])
		}

		// Verify author field exists
		_, ok = comment["author"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected author object for comment %d, got %v", i, comment["author"])
		}

		// Verify timestamps
		_, ok = comment["createdAt"].(string)
		if !ok {
			t.Fatalf("expected string createdAt for comment %d, got %v", i, comment["createdAt"])
		}

		_, ok = comment["updatedAt"].(string)
		if !ok {
			t.Fatalf("expected string updatedAt for comment %d, got %v", i, comment["updatedAt"])
		}
	}

	// Verify comment order and content
	comment1, _ := comments[0].(map[string]interface{})
	comment2, _ := comments[1].(map[string]interface{})
	comment3, _ := comments[2].(map[string]interface{})

	if comment1["body"] != "First comment" {
		t.Fatalf("expected first comment body 'First comment', got %v", comment1["body"])
	}

	if comment2["body"] != "Second comment" {
		t.Fatalf("expected second comment body 'Second comment', got %v", comment2["body"])
	}

	if comment3["body"] != "Third comment" {
		t.Fatalf("expected third comment body 'Third comment', got %v", comment3["body"])
	}
}

func TestGetCommentsEmpty(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article without comments
	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"No Comments","description":"desc","body":"body"}}`)

	// Get comments for article with no comments
	resp := performGetCommentsRequest(t, slug, "Token "+common.GenToken(author.ID))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	commentsPayload, ok := payload["comments"]
	if !ok {
		t.Fatalf("expected comments field, got: %s", resp.Body.String())
	}

	// Comments can be nil or an empty array
	if commentsPayload != nil {
		comments, ok := commentsPayload.([]interface{})
		if !ok {
			t.Fatalf("expected comments to be array, got: %T", commentsPayload)
		}

		if len(comments) != 0 {
			t.Fatalf("expected 0 comments, got %d", len(comments))
		}
	}
}

func TestGetCommentsArticleNotFound(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Get comments for non-existent article
	resp := performGetCommentsRequest(t, "non-existent-article", "Token "+common.GenToken(author.ID))

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

func TestDeleteCommentSuccess(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article and a comment
	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Delete Comment","description":"desc","body":"body"}}`)

	// Create a comment
	createResp := performCreateCommentRequest(t, slug, "Token "+common.GenToken(author.ID), `{"comment":{"body":"Comment to delete"}}`)

	if createResp.Code != http.StatusCreated {
		t.Fatalf("failed to create comment: status %d", createResp.Code)
	}

	var createPayload map[string]map[string]interface{}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("failed to parse create response json: %v", err)
	}

	commentPayload, ok := createPayload["comment"]
	if !ok {
		t.Fatalf("expected comment payload in create response, got: %s", createResp.Body.String())
	}

	commentID := int64(commentPayload["id"].(float64))
	commentIDStr := strconv.FormatInt(commentID, 10)

	// Verify comment exists before deletion
	getRes := performGetCommentsRequest(t, slug, "Token "+common.GenToken(author.ID))
	var getPayload map[string]interface{}
	if err := json.Unmarshal(getRes.Body.Bytes(), &getPayload); err != nil {
		t.Fatalf("failed to parse get response json: %v", err)
	}

	commentsBeforeDelete := getPayload["comments"]
	if commentsBeforeDelete != nil {
		comments := commentsBeforeDelete.([]interface{})
		if len(comments) != 1 {
			t.Fatalf("expected 1 comment before deletion, got %d", len(comments))
		}
	}

	// Delete the comment
	deleteResp := performDeleteCommentRequest(t, slug, commentIDStr, "Token "+common.GenToken(author.ID))

	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusNoContent, deleteResp.Code, deleteResp.Body.String())
	}

	// Verify comment is deleted
	var emptyCommentModel CommentModel
	if err := common.DB.First(&emptyCommentModel, commentID).Error; err == nil {
		t.Fatalf("expected comment to be deleted from database, but found: %v", emptyCommentModel)
	}

	// Verify comment list is now empty
	getAfterDeleteRes := performGetCommentsRequest(t, slug, "Token "+common.GenToken(author.ID))
	var getAfterDeletePayload map[string]interface{}
	if err := json.Unmarshal(getAfterDeleteRes.Body.Bytes(), &getAfterDeletePayload); err != nil {
		t.Fatalf("failed to parse get response json after delete: %v", err)
	}

	commentsAfterDelete := getAfterDeletePayload["comments"]
	if commentsAfterDelete != nil {
		comments := commentsAfterDelete.([]interface{})
		if len(comments) != 0 {
			t.Fatalf("expected 0 comments after deletion, got %d", len(comments))
		}
	}
}

func TestDeleteCommentUnauthorized(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article and a comment
	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Unauthorized Delete","description":"desc","body":"body"}}`)

	createResp := performCreateCommentRequest(t, slug, "Token "+common.GenToken(author.ID), `{"comment":{"body":"Comment"}}`)

	var createPayload map[string]map[string]interface{}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createPayload); err != nil {
		t.Fatalf("failed to parse create response json: %v", err)
	}

	commentPayload, ok := createPayload["comment"]
	if !ok {
		t.Fatalf("expected comment payload in create response, got: %s", createResp.Body.String())
	}

	commentID := int64(commentPayload["id"].(float64))
	commentIDStr := strconv.FormatInt(commentID, 10)

	// Try to delete without authorization
	deleteResp := performDeleteCommentRequest(t, slug, commentIDStr, "")

	if deleteResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, deleteResp.Code, deleteResp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(deleteResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object, got: %s", deleteResp.Body.String())
	}

	if errorsPayload["auth"] != users.ErrUnauthorized.Error() {
		t.Fatalf("expected auth error %q, got %v", users.ErrUnauthorized.Error(), errorsPayload["auth"])
	}
}

func TestDeleteCommentInvalidID(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article
	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Invalid ID","description":"desc","body":"body"}}`)

	// Try to delete comment with invalid ID
	deleteResp := performDeleteCommentRequest(t, slug, "invalid-id", "Token "+common.GenToken(author.ID))

	if deleteResp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusBadRequest, deleteResp.Code, deleteResp.Body.String())
	}

	var payload map[string]map[string]interface{}
	if err := json.Unmarshal(deleteResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response json: %v", err)
	}

	errorsPayload, ok := payload["errors"]
	if !ok {
		t.Fatalf("expected errors object, got: %s", deleteResp.Body.String())
	}

	if _, exists := errorsPayload["invalid_id"]; !exists {
		t.Fatalf("expected invalid_id error, got: %v", errorsPayload)
	}
}

func TestDeleteCommentNotFound(t *testing.T) {
	setupArticleTestDB(t)
	author := seedArticleAuthor(t)

	// Create an article
	slug := createArticleAndReturnSlug(t, author.ID, `{"article":{"title":"Delete Non-existent","description":"desc","body":"body"}}`)

	// Try to delete non-existent comment (idempotent - returns success without error)
	deleteResp := performDeleteCommentRequest(t, slug, "9999", "Token "+common.GenToken(author.ID))

	// The implementation allows idempotent deletes - returns 204 even if comment doesn't exist
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusNoContent, deleteResp.Code, deleteResp.Body.String())
	}
}
