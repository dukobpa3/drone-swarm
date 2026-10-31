.PHONY: test lint sonar

# Go modules in the monorepo. Each one is tested and linted from its own
# directory; add a new service's module here.
GO_MODULES := sim

test:
	@for m in $(GO_MODULES); do (cd $$m && go test ./...) || exit 1; done

lint:
	@for m in $(GO_MODULES); do (cd $$m && golangci-lint run ./...) || exit 1; done

# Tests with coverage, golangci-lint's reports, the scan of this checkout and the
# quality gate (red: a non-zero exit). Lint findings do not stop it: Sonar imports
# them and the gate decides. SONAR_HOST_URL and SONAR_TOKEN come from the environment
# or .env (a worktree uses the main checkout's). Coverage paths are rewritten from the
# Go module path to the module's directory, since go.mod is not at the project root.
# scripts/sonar-scope says whether this is the main branch, a pull request or another
# branch. SONAR_GATE=false sends the report without waiting for the gate.
SONAR_GATE ?= true

sonar:
	@env=.env; [ -f $$env ] || env="$$(git rev-parse --path-format=absolute --git-common-dir)/../.env"; \
	set -a; [ ! -f "$$env" ] || . "$$env"; set +a; \
	test -n "$$SONAR_HOST_URL" && test -n "$$SONAR_TOKEN" || { echo 'sonar: set SONAR_HOST_URL and SONAR_TOKEN (.env)' >&2; exit 1; }; \
	scope=$$(scripts/sonar-scope) || exit 1; \
	echo "sonar: $${scope:-main branch}"; \
	for m in $(GO_MODULES); do \
	  (cd $$m && go test -coverpkg=./... -coverprofile=coverage.out ./... && \
	   mod=$$(go list -m) && sed "s|^$$mod/|$$m/|" coverage.out > coverage.tmp && mv coverage.tmp coverage.out) || exit 1; \
	  (cd $$m && golangci-lint run --issues-exit-code=0 --output.text.path=stderr \
	     --output.checkstyle.path=$(CURDIR)/$$m/golangci-lint.xml ./... 2> /dev/null) || exit 1; \
	done; \
	sonar-scanner -Dsonar.qualitygate.wait=$(SONAR_GATE) $$scope
