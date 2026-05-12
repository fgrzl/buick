//go:build !integration

package integration

import "testing"

func TestShouldSkipDockerBackedSuitesGivenDefaultBuildTagsWhenIntegrationTagMissing(t *testing.T) {
	t.Skip(`docker-backed tests are disabled without -tags=integration.

Bring up the stack from the repo root (nginx + buickd; see compose.yml):
  docker compose up -d --build

Then run:
  BUICK_INTEGRATION=1 go test -tags=integration ./tests/integration/...`)
}
