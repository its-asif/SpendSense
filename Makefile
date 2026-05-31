OPENAPI_SPEC := backend/internal/httpapi/openapi.yaml
OPENAPI_OUT := backend/internal/httpapi/openapi/types.gen.go
OPENAPI_GEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1

.PHONY: openapi test

openapi:
	cd backend && mkdir -p internal/httpapi/openapi && $(OPENAPI_GEN) --generate types -package openapi -o internal/httpapi/openapi/types.gen.go internal/httpapi/openapi.yaml

test:
	cd backend && go test ./...
