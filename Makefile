.PHONY: build test run vet lint tidy docker-build docker-up docker-down docker-logs migrate

build:
	go build -o ./bin/server ./cmd/server

test:
	go test -race -count=1 -timeout 30s ./...

run: build
	APP_ENV=local ./bin/server

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	GONOSUMDB='github.com/cccvno1/goplate/*' GONOSUMCHECK='github.com/cccvno1/goplate/*' go mod tidy

docker-build:
	docker build -t ledger-agent .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f --tail=50

migrate:
	@for f in migrations/postgres/*.sql; do \
		echo "Applying $$f..."; \
		docker exec -i ledger-agent-postgres-1 psql -U ledger -d ledger < "$$f" || exit 1; \
	done

