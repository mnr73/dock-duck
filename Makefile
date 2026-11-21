.PHONY: dev build test lint web server

web:
	cd apps/web && pnpm install && pnpm build
	rm -rf apps/server/internal/webui/dist/* && cp -r apps/web/dist/. apps/server/internal/webui/dist/ && touch apps/server/internal/webui/dist/.gitkeep

server:
	cd apps/server && go build -o ../../bin/dockduck ./cmd/dockduck

build: web server

dev:
	@echo "run in two terminals: (cd apps/server && go run ./cmd/dockduck --dev) and (cd apps/web && pnpm dev)"

test:
	cd apps/server && go test ./...
	cd apps/web && pnpm test

lint:
	cd apps/server && go vet ./...
	cd apps/web && pnpm lint
