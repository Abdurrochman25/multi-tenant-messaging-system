#!make

generate-model:
	@echo "generate model"
	@godotenv -f .env sqlboiler psql

run:
	go run main.go
