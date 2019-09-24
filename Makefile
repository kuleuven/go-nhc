all: build complete

build:
	go build -mod vendor -v .

complete: build
	./go-nhc --completion-script-bash > go-nhc.bash_completion
	./go-nhc --completion-script-zsh > go-nhc.zsh_completion
	./go-nhc --help-man | gzip > go-nhc.1.gz
