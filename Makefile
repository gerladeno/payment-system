start_db:
	docker-compose up -d

stop_db:
	docker-compose down

tests: start_db
	go test -race -count=1 tests/store/store_test.go
	go test -race -count=1 tests/rest/handler_test.go

run: start_db
	PG_DSN="postgresql://user:user_pw@localhost:5433/payments?sslmode=disable" go run cmd/payments/main.go