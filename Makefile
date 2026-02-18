.PHONY: build test test-server test-mobile lint lint-server lint-mobile lint-admin clean dev-server proto admin-ui cross-compile

# Build server binary with embedded admin UI
build: admin-ui
	cd server && go build -o ../build/sovereign ./cmd/sovereign
	cd server && go build -o ../build/sovereign-cli ./cmd/sovereign-cli

# Build admin UI (output to server/web/dist/)
admin-ui:
	cd admin-ui && npm install --silent && npm run build

# Run all tests
test: test-server test-mobile

test-server:
	cd server && go test ./...

test-mobile:
	cd mobile && npm test -- --passWithNoTests

# Lint all code
lint: lint-server lint-mobile lint-admin

lint-server:
	cd server && go vet ./...

lint-mobile:
	cd mobile && npx tsc --noEmit

lint-admin:
	cd admin-ui && npx tsc --noEmit

# Remove build artifacts
clean:
	rm -rf build/
	rm -rf server/web/dist/*
	rm -rf admin-ui/node_modules/.vite

# Run server in development mode
dev-server:
	cd server && go run ./cmd/sovereign

# Regenerate protobuf stubs
proto:
	protoc --go_out=server/internal/protocol --go_opt=paths=source_relative \
		protocol/messages.proto

# Cross-compile for multiple platforms
cross-compile: admin-ui
	mkdir -p build
	GOOS=linux GOARCH=amd64 cd server && go build -o ../build/sovereign-linux-amd64 ./cmd/sovereign
	GOOS=linux GOARCH=arm64 cd server && go build -o ../build/sovereign-linux-arm64 ./cmd/sovereign
	GOOS=darwin GOARCH=amd64 cd server && go build -o ../build/sovereign-darwin-amd64 ./cmd/sovereign
	GOOS=darwin GOARCH=arm64 cd server && go build -o ../build/sovereign-darwin-arm64 ./cmd/sovereign
