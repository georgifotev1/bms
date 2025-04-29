include .env

migrateup:
	@goose -dir internal/sql/schema postgres "postgresql://${DB_DOCKER_USER}:${DB_DOCKER_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable" up

migratedown:
	@goose -dir internal/sql/schema postgres "postgresql://${DB_DOCKER_USER}:${DB_DOCKER_PASSWORD}@localhost:5432/${DB_NAME}?sslmode=disable" down

test:
	@go test -v ./...

create_db:
	docker-compose exec db createdb --username=${DB_DOCKER_USER} --owner=${DB_DOCKER_USER} ${DB_NAME}

gen-docs:
	@swag init -g /main.go -d cmd && swag fmt

queries:
	sqlc generate && mockery
