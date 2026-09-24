# ローカルでの品質チェック。lint ツールは tools/go.mod で版を固定し、本体の go.mod には含めない。
GO    ?= go
GOFMT := $(shell $(GO) env GOROOT)/bin/gofmt
TOOL  := $(GO) tool -modfile=tools/go.mod

.PHONY: check fmt fmt-check vet lint vuln test golden

check: fmt-check vet lint vuln test

fmt:
	$(GOFMT) -s -w .

# PATH 上の古い gofmt はメソッドの型パラメータ (Go 1.27) を解釈できないため、ツールチェーン同梱のものを使う。
fmt-check:
	@out="$$($(GOFMT) -s -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet ./...
	GOOS=windows $(GO) vet ./...

lint:
	$(TOOL) golangci-lint run ./...
	$(TOOL) staticcheck -checks all ./...

vuln:
	$(TOOL) govulncheck ./...

test:
	$(GO) test -race ./...

# sudachi.rs との互換性。SudachiDict core 20250515 の system_core.dic が必要。
golden:
	@test -n "$(SUDACHIN_TEST_DICT)" || { echo "set SUDACHIN_TEST_DICT to SudachiDict core 20250515 system_core.dic"; exit 1; }
	SUDACHIN_TEST_DICT=$(SUDACHIN_TEST_DICT) $(GO) test -count=1 -run Golden .
