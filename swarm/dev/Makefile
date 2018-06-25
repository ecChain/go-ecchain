.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/ecchain/go-ecchain/cmd/swarm
	go build -o bin/gec     github.com/ecchain/go-ecchain/cmd/gec
	go build -o bin/bootnode github.com/ecchain/go-ecchain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/ecchain/go-ecchain/swarm/...
