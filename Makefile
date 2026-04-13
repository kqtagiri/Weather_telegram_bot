include .env
export

start_program:
	@go run main.go

start_docker:
	@docker compose up -d --build

end_docker:
	@docker compose down