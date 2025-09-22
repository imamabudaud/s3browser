local-run:
	@echo "Starting local go app"
	@go run cmd/main.go

docker-build:
	@echo "Building Docker image..."
	@docker build -t s3browser .

docker-run:
	@echo "Starting with docker-compose..."
	@docker-compose up -d

docker-stop:
	@echo "Stopping docker-compose..."
	@docker-compose down

docker-logs:
	@echo "Showing docker-compose logs..."
	@docker-compose logs -f

docker-status:
	@echo "Showing docker-compose status..."
	@docker-compose ps

