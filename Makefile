#!make

generate-model:
	@echo "generate model"
	@godotenv -f .env sqlboiler psql

swagger:
	swag init -g cmd/api/main.
	
test:
	go test -v ./integration_test.go -timeout=5m

run:
	go run main.go
