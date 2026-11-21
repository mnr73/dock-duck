.PHONY: dev build test lint web server

web:
	pnpm install --frozen-lockfile
	cd apps/web && pnpm build
	rm -rf apps/server/internal/webui/dist/* && cp -r apps/web/dist/. apps/server/internal/webui/dist/ && touch apps/server/internal/webui/dist/.gitkeep

server:
	cd apps/server && go build -trimpath -ldflags "-s -w -X main.version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o ../../bin/dockduck ./cmd/dockduck

build: web server

dev:
	@echo "run in two terminals: 'cd apps/server && go run ./cmd/dockduck --dev'  and  'cd apps/web && pnpm dev' (UI on :5173, proxies /api to :8080)"

test:
	cd apps/server && go test ./...
	cd apps/web && pnpm test

lint:
	cd apps/server && go vet ./...
	cd apps/web && pnpm lint
