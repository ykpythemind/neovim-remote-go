package main

import (
	"os"

	nv "github.com/ykpythemind/neovim-remote-go"
)

func main() {
	nv.Run(os.Stdout, os.Args...)
}
