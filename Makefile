OPENAPI_SPEC := backend/internal/httpapi/openapi.yaml
OPENAPI_OUT := backend/internal/httpapi/openapi/types.gen.go
OPENAPI_GEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1

.PHONY: openapi test cli-build cli-run

openapi:
	cd backend && mkdir -p internal/httpapi/openapi && $(OPENAPI_GEN) --generate types -package openapi -o internal/httpapi/openapi/types.gen.go internal/httpapi/openapi.yaml

test:
	cd backend && go test ./...

test-cover:
	cd backend && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

cover-service:
	cd backend && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | grep "service.go"

cli-build:
	mkdir -p bin
	cd cli && go build -o ../bin/spendsense .

cli-run: cli-build
	./bin/spendsense --help
