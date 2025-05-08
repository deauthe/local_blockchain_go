build:
	go build -o ./bin/projectx

prod_run:
	"go build . -o ./bin/projectx && ./bin/projectx"

run:
	"rm -rf ./data/blocks.gob | go build -o ./bin/projectx"

test:
	go test ./tests/ -v