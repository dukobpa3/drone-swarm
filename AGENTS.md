# drone-swarm

Monorepo for the swarm: ground control, companion, protocols and the
simulation launcher. `sim/` is developed here; github.com/dukobpa3/drone-swarm-sim
is only its mirror, updated with `make sim-push` after a merge to `master`. Never
change the mirror directly.

## Before committing

- Go code builds and vets (`cd sim && go vet ./... && go build ./...`).
- `make lint` (golangci-lint, `.golangci.yml`) adds nothing new; the pre-commit hook runs it.
  Answer a false positive on its line with `//nolint:<linter> // <why>` or
  `// NOSONAR <why>`, never by closing the issue on the server.
- SonarQube, when `SONAR_HOST_URL` and `SONAR_TOKEN` are set: the `sonar` skill
  ([`.agents/skills/sonar`](.agents/skills/sonar/SKILL.md)) says how. Without them, skip it:
  it is a check on the maintainer's machine. Never print the token.
- A new service's code directories go into `sonar.sources`/`sonar.tests` in
  `sonar-project.properties` and into `COVERED` in `.githooks/pre-push`.
  A new Go module also goes into `GO_MODULES` in the Makefile and its
  `<module>/golangci-lint.xml` into `sonar.go.golangci-lint.reportPaths`.
