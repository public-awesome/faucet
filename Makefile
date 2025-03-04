build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/faucet-linux *.go

build-mac:
	GOOS=darwin GOARCH=amd64 go build -o bin/faucet-mac main.go

build:
	go build -o bin/faucet-server main.go
