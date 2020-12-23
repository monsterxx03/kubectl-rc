all: build_rc build_sen

all_arch: build_rc_linux build_rc_darwin build_sen_linux build_sen_darwin

build_rc:
	go build -o bin/kubectl-rc ./cmd/rc 

build_rc_linux:
	GOOS=linux GOARCH=amd64 go build -o bin/kubectl-rc_linux_amd64  ./cmd/rc


build_rc_darwin:
	GOOS=darwin GOARCH=amd64 go build -o bin/kubectl-rc_darwin_amd64  ./cmd/rc


build_sen:
	go build -o bin/kubectl-sen ./cmd/sen

build_sen_linux:
	GOOS=linux GOARCH=amd64 go build -o bin/kubectl-sen_linux_amd64  ./cmd/sen


build_sen_darwin:
	GOOS=darwin GOARCH=amd64 go build -o bin/kubectl-sen_darwin_amd64  ./cmd/sen
