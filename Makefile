.PHONY: test sonar

test:
	cd sim && go test ./...

# Tests with coverage, SonarQube scan and quality gate; a red gate fails.
# Needs SONAR_HOST_URL and SONAR_TOKEN, from the environment or .env.
sonar:
	@set -a; [ ! -f .env ] || . ./.env; set +a; \
	test -n "$$SONAR_HOST_URL" && test -n "$$SONAR_TOKEN" || { echo 'Set SONAR_HOST_URL and SONAR_TOKEN (see CONTRIBUTING.md)'; exit 1; }; \
	(cd sim && go test -coverprofile=coverage.out ./... && \
	 mod=$$(go list -m) && sed "s|^$$mod/|sim/|" coverage.out > coverage.tmp && mv coverage.tmp coverage.out) && \
	sonar-scanner -Dsonar.qualitygate.wait=true
