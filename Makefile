build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/faucet-linux cmd/*.go

build-mac:
	GOOS=darwin GOARCH=amd64 go build -o bin/faucet-mac cmd/*.go

build:
	go build -o bin/faucet-server cmd/*.go

build-docker: build-linux
	docker build -t publicawesome/faucet:latest .
