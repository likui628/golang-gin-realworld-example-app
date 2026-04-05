/*
Package articles contains the article management and article-facing HTTP layer.

models.go: GORM-backed article model only

repository.go: typed persistence boundary for article lookups and writes

service.go: article creation, update, deletion, and retrieval business logic

handler.go: Gin HTTP handlers with injected service dependencies

routers.go: Gin route registration only

serializers.go: response DTOs for article payloads

validators.go: request binding and validation for article endpoints

unit_test.go: article creation, update, deletion, and retrieval regression tests
*/
package articles
