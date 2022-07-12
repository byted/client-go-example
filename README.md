## client-go-example Project

- `brew install golang`
- `go mod init github.com/byted/client-go-example/client`
- `go get k8s.io/client-go@latest`

## Local K8 Cluster

- `brew install docker`
- `go install sigs.k8s.io/kind@v0.14.0`
- `export PATH=$PATH:<GOPATH>/bin` (default:` ~/go`)
- (`source <(kubectl completion zsh)` to add autocomplete)
- `kind create cluster --config=cluster/cluster-config.yaml`