.PHONY: tests, mocks

tests:
	go test -v -count 1 -race ./...

mocks:
	mockery --dir "transport" --name Transport --output ./mocks --filename transport.go
