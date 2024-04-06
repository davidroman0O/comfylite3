
sqlite3:
	go test -v -count=1 -run ^TestMemory$$ .
	go test -v -count=1 -run ^TestFile$$ .
	go test -v -count=1 -run ^TestLockedGist$$ .
	go test -v -count=1 ./test

comfy:
	go test -v -count=1 ./test