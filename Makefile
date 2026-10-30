.PHONY: test lint sonar

# Go modules in the monorepo. Each one is tested and linted from its own
# directory; add a new service's module here.
GO_MODULES := sim

test:
	@for m in $(GO_MODULES); do (cd $$m && go test ./...) || exit 1; done

lint:
	@for m in $(GO_MODULES); do (cd $$m && golangci-lint run ./...) || exit 1; done

# Tests with coverage, golangci-lint reports, SonarQube scan and quality
# gate; a red gate fails. Lint findings do not stop it: Sonar imports them
# and the gate decides. Needs SONAR_HOST_URL and SONAR_TOKEN, from the
# environment or .env. Coverage paths are rewritten from the Go module path
# to the module's directory, since go.mod is not at the project root.
sonar:
	@set -a; [ ! -f .env ] || . ./.env; set +a; \
	test -n "$$SONAR_HOST_URL" && test -n "$$SONAR_TOKEN" || { echo 'Set SONAR_HOST_URL and SONAR_TOKEN (see CONTRIBUTING.md)'; exit 1; }; \
	for m in $(GO_MODULES); do \
	  (cd $$m && go test -coverprofile=coverage.out ./... && \
	   mod=$$(go list -m) && sed "s|^$$mod/|$$m/|" coverage.out > coverage.tmp && mv coverage.tmp coverage.out) || exit 1; \
	  (cd $$m && golangci-lint run --issues-exit-code=0 --output.text.path=stderr \
	     --output.checkstyle.path=$(CURDIR)/$$m/golangci-lint.xml ./... 2> /dev/null) || exit 1; \
	done; \
	sonar-scanner -Dsonar.qualitygate.wait=true
