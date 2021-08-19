start_db:
	docker-compose up -d pg

stop:
	docker-compose down

tests: start_db
	go test -race -count=1 tests/store/store_test.go
	go test -race -count=1 tests/rest/handler_test.go

build_locally:
	go build cmd/payments/main.go

run_from_code: start_db
	PG_DSN="postgresql://user:user_pw@localhost:5433/payments?sslmode=disable" go run cmd/payments/main.go

build:
	docker-compose build

run: build
	docker-compose up -d

up:
	docker-compose up -d

integration-test: run
	go mod download
	go test -race -count=1 tests/integration/integration_test.go
	docker-compose down