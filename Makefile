FLAGS="-X main.BuildTime=$$(date -u +'%Y-%m-%dT%H:%M:%SZ') -X main.GitCommit=$$(git rev-parse HEAD)"


build-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags=$(FLAGS) -o ./downloads/linux_armv7 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags=$(FLAGS) -o ./downloads/linux_arm64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags=$(FLAGS) -o ./downloads/linux_amd64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags=$(FLAGS) -o ./downloads/darwin_arm64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags=$(FLAGS) -o ./downloads/darwin_amd64 main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags=$(FLAGS) -o ./downloads/windows_amd64.exe main.go

