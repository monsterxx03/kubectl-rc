build:
	go build -o bin/kubectl-rc

build_linux:
	GOOS=linux GOARCH=amd64 go build -o bin/kubectl-rc_linux_amd64


build_darwin:
	GOOS=darwin GOARCH=amd64 go build -o bin/kubectl-rc_darwin_amd64
