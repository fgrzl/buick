// Package integration holds docker-compose-backed integration tests for buickd.
//
// From the repository root, start the stack (nginx backends + buickd; see ../../compose.yml
// and ../../compose.buick.yml), then run:
//
//	BUICK_INTEGRATION=1 go test -tags=integration ./tests/integration/...
//	docker compose down
//
// Default `go test ./...` compiles this package but skips docker-backed tests.
package integration
