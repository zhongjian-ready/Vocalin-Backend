.PHONY: run build seed swagger fmt tidy test

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

seed:
	go run cmd/seeder/main.go

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go --parseDependency --parseInternal

fmt:
	gofmt -w cmd internal pkg

tidy:
	go mod tidy

test:
	go test ./...
