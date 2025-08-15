#!make

generate-model:
	@echo "generate model"
	@godotenv -f .env sqlboiler psql

swagger:
	swag init -g cmd/api/main.go

run:
	go run main.go
