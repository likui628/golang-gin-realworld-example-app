package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/likui628/golang-gin-realworld-example-app/articles"
	"github.com/likui628/golang-gin-realworld-example-app/common"
	"github.com/likui628/golang-gin-realworld-example-app/users"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbFile := filepath.Join(t.TempDir(), "app.db")
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			sqlDB.Close()
		}
	})

	return db
}

func TestMigrateCreatesTables(t *testing.T) {
	db := openTestDB(t)

	Migrate(db)

	for _, table := range []interface{}{
		&users.UserModel{},
		&users.FollowModel{},
		&articles.TagModel{},
		&articles.ArticleModel{},
		&articles.FavoriteModel{},
		&articles.CommentModel{},
	} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table for %T to exist", table)
		}
	}
}

func TestSetupRouterRegistersCoreRoutes(t *testing.T) {
	t.Setenv(common.JWTSecretEnvVar, "test-secret")
	db := openTestDB(t)
	Migrate(db)

	service := users.NewUserService(users.NewUserRepository(db))
	registered, err := service.Register(users.RegisterUserInput{
		Username: "routeruser",
		Email:    "router@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to seed user for router test: %v", err)
	}

	router := setupRouter(db)

	tagsRequest := httptest.NewRequest(http.MethodGet, "/api/tags", nil)
	tagsResponse := httptest.NewRecorder()
	router.ServeHTTP(tagsResponse, tagsRequest)
	if tagsResponse.Code != http.StatusOK {
		t.Fatalf("expected /api/tags to return %d, got %d", http.StatusOK, tagsResponse.Code)
	}

	unauthorizedRequest := httptest.NewRequest(http.MethodGet, "/api/user", nil)
	unauthorizedResponse := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedResponse, unauthorizedRequest)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected /api/user without auth to return %d, got %d", http.StatusUnauthorized, unauthorizedResponse.Code)
	}

	authorizedRequest := httptest.NewRequest(http.MethodGet, "/api/user", nil)
	authorizedRequest.Header.Set("Authorization", "Token "+registered.Token)
	authorizedResponse := httptest.NewRecorder()
	router.ServeHTTP(authorizedResponse, authorizedRequest)
	if authorizedResponse.Code != http.StatusOK {
		t.Fatalf("expected /api/user with auth to return %d, got %d, body: %s", http.StatusOK, authorizedResponse.Code, authorizedResponse.Body.String())
	}
}

func TestCloseDBClosesConnection(t *testing.T) {
	db := openTestDB(t)

	closeDB(db)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("expected sql.DB handle, got %v", err)
	}
	if err := sqlDB.Ping(); err == nil {
		t.Fatal("expected ping to fail after closeDB")
	}
}
