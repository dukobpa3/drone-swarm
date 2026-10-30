# drone-swarm

Monorepo for the swarm: ground control, companion, protocols and the
simulation launcher. `sim/` is a `git subtree` of
github.com/dukobpa3/drone-swarm-sim; change it there or push it back with
`git subtree push --prefix=sim git@github.com:dukobpa3/drone-swarm-sim.git master`.

## Before committing

- Go code builds and vets (`cd sim && go vet ./... && go build ./...`).
- `make lint` (golangci-lint, `.golangci.yml`) adds nothing new. Answer a
  false positive on its line with `//nolint:<linter> // <why>` or
  `// NOSONAR <why>`, never by closing the issue on the server.
- If `SONAR_HOST_URL` and `SONAR_TOKEN` are set (environment or `.env`):
  `make sonar` passes (quality gate green). See what it flagged with
  `sonar list issues --format table`. If they are not set, skip this: it is
  a check on the maintainer's machine.
- A new service's code directories go into `sonar.sources`/`sonar.tests` in
  `sonar-project.properties` and into the pattern in `.githooks/pre-commit`.
  A new Go module also goes into `GO_MODULES` in the Makefile and its
  `<module>/golangci-lint.xml` into `sonar.go.golangci-lint.reportPaths`.

## SonarQube rules

- Never print, log, commit or ask for the value of `SONAR_TOKEN`; only check
  that it is set.
- Do not change server settings or quality profiles, and do not close issues
  or mark them false positive on the server, unless asked.
- Do not run `sonar integrate` (it downloads a binary and can overwrite hooks).
- Do not commit with `--no-verify` unless asked, and say so in the message.
