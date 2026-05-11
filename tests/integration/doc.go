// Package integration holds docker-compose-backed integration tests for buickd.
// Run the stack from the repository root:
//
//	docker compose up -d --build
//	BUICK_INTEGRATION=1 go test -tags=integration ./tests/integration/...
//	docker compose down
//
// Default `go test ./...` compiles this package but skips docker-backed tests.
package integration
