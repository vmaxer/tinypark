.PHONY: run win64 clean

NAME = tinypark

$(NAME): go.mod main.go pkg/adventure/*.go
	go build -o $@ .

run: $(NAME)
	./$(NAME)

win64:
	GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o $(NAME).exe .
	@echo "$(NAME).exe is ready to be shared with Windows users"

clean:
	rm -f $(NAME) $(NAME).exe
