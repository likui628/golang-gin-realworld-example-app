/*
Package users contains the user authentication and user-facing HTTP layer.

models.go: GORM-backed user model and password helpers

repository.go: typed persistence boundary for user lookups and writes

service.go: registration, login, and user retrieval business logic

routers.go: Gin route registration and thin HTTP handlers

middlewares.go: JWT auth middleware and current-user context access

serializers.go: response DTOs for user payloads

validators.go: request binding and validation for user endpoints

unit_test.go: registration, login, and auth middleware regression tests
*/
package users
