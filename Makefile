BINARY  := crunch
CMD     := ./cmd/crunch
LDFLAGS := -s -w

.PHONY: all build tidy test vet fmt run clean

all: build

# -trimpath убирает пути сборки из бинарника,
# -s -w убирает таблицы символов и debug-информацию
build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) $(CMD)

tidy:
	go mod tidy

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)