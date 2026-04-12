# ![RealWorld Example App](logo.png)

> ### Golang/Gin codebase containing real world examples (CRUD, auth, advanced patterns, etc) that adheres to the [RealWorld](https://github.com/gothinkster/realworld) spec and API.


This codebase was created to demonstrate a fully fledged fullstack application built with **Golang/Gin** including CRUD operations, authentication, routing, pagination, and more.

## Architecture
This project follows a layered architecture with strict one-way dependencies:

Handler → Service → Repository → Database

Each layer has a single responsibility and depends only on the layer below it.

| File | Responsibility |
|---|---|
| `models.go` | GORM schema definition only |
| `repository.go` | Database access interface and GORM implementation |
| `service.go` | Business logic |
| `handler.go` | HTTP request binding and response, receives injected service |
| `routers.go` | Route registration only |

Dependencies are wired once at startup in `main.go`:
```go
userRepository := users.NewUserRepository(db)
userService    := users.NewUserService(userRepository)
userHandler    := users.NewUserHandler(userService)
```

No layer constructs its own dependencies at request time.

## Environment Config

Environment variables can be set directly in your shell or via a .env file.
Available environment variables:
```
PORT=8080                     # Server port (default: 8080)
GIN_MODE=debug               # Gin mode: debug or release
DB_PATH=./data/gorm.db       # SQLite database path (default: ./data/gorm.db)
JWT_SECRET=replace-me        # Required secret used to sign and verify JWT tokens
```

See .env.example for a complete template.

## Install Dependencies
This project targets Go 1.26.1 as declared in `go.mod`.

1. Install Go 1.26.1 or a compatible newer version.
2. Create a local environment file from the example.
3. Download the Go module dependencies.

```powershell
Copy-Item .env.example .env
go mod download
```

Required note:
Set `JWT_SECRET` in `.env` before starting the server, otherwise authentication tokens should not be considered secure.

SQLite note:
This project uses `github.com/glebarez/sqlite`, so it does not require CGO or a local GCC toolchain to run on Windows.

## Run the Server
After installing dependencies and configuring `.env`, start the API from the project root:

```powershell
go run .
```

The server reads `.env` on startup, runs the database migration automatically, and listens on `PORT` (default: `8080`).

Once it is running, the API is available at:

```text
http://localhost:8080/api
```

If you want to use a different port for local development:

```powershell
$env:PORT = "3000"
go run .
```

## Testing
From the project root, run:
```powershell
go test ./...
```

## Test Coverage
Current test coverage (2026):
* Total: 71.9%
* articles: 71.5% 
* users: 70.1% 
* common: 86.4% 

Run coverage report:

```powershell
go test -coverprofile='coverage.out' ./...
go tool cover -func='coverage.out'
```