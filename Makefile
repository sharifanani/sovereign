.PHONY: build test test-server test-mobile lint lint-server lint-mobile lint-admin clean dev-server dev-mobile dev proto admin-ui cross-compile stop

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

# Run server in development mode (foreground)
dev-server:
	cd server && go run ./cmd/sovereign

# Run mobile app in web browser (foreground)
dev-mobile:
	cd mobile && npx expo start --web --port 19006

# Run both server and mobile for development (background, use `make stop` to shut down)
dev: build
	@echo "Starting Sovereign server on :8080..."
	@./build/sovereign & echo $$! > .server.pid
	@sleep 1
	@echo "Starting Expo web on :19006..."
	@cd mobile && npx expo start --web --port 19006 & echo $$! > .expo.pid
	@echo ""
	@echo "Sovereign is running:"
	@echo "  Server:  http://localhost:8080"
	@echo "  Mobile:  http://localhost:19006"
	@echo "  Admin:   http://localhost:8080/admin/"
	@echo ""
	@echo "Run 'make stop' to shut everything down."

# Stop all dev processes
stop:
	@if [ -f .server.pid ]; then kill $$(cat .server.pid) 2>/dev/null; rm -f .server.pid; echo "Server stopped."; fi
	@if [ -f .expo.pid ]; then kill $$(cat .expo.pid) 2>/dev/null; rm -f .expo.pid; echo "Expo stopped."; fi
	@pkill -f "build/sovereign" 2>/dev/null || true
	@pkill -f "expo start" 2>/dev/null || true

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
