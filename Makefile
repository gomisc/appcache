gomod:
	go mod tidy -go=1.16 && go mod tidy -go=1.17
	go mod download
