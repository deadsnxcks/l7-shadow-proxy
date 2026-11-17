VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)
CONFIG ?= configs/local.yaml
BACKENDS = 9201:default 9202:api-v1 9203:api-v2 9204:users 9205:health

.PHONY: proxy run test bench backends stop-backends clean

proxy:
	go build -ldflags "$(LDFLAGS)" -o bin/proxy ./cmd/proxy

run: proxy
	bin/proxy -config $(CONFIG)

test:
	go test -race -count=1 ./...

bench:
	go test -run '^$$' -bench=. -benchmem ./...

backends: stop-backends
	go build -o bin/backend ./cmd/backend
	for b in $(BACKENDS); do \
		bin/backend -port $${b%%:*} -name $${b##*:} & \
	done

stop-backends:
	-pkill -f bin/backend

clean: stop-backends
	rm -rf bin