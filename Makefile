generate:
	go generate ./internal/handler/

test:
	go test ./... -v

install:
	go install .
