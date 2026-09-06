# go-particle-lab
# Go の物理エンジンを WebAssembly に組み、web/ を静的配信する。

GO ?= go
WASM = web/main.wasm

.PHONY: all wasm test vet serve clean

all: test wasm

wasm:
	GOOS=js GOARCH=wasm $(GO) build -trimpath -ldflags="-s -w" -o $(WASM) ./cmd/wasm

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

serve:
	python -m http.server 8080 --directory web

clean:
	rm -f $(WASM)
