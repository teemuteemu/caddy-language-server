test:
	go test ./... -v

install:
	go build -o ~/bin/caddy-ls ./cmd/caddy-ls
