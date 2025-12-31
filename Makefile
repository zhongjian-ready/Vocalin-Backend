.PHONY: run build swagger

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

seed:
	go run cmd/seeder/main.go

swagger:
	swag init -g cmd/server/main.go
