package main

import "github.com/LucienZhang/goto/cmd"

func main() {
	cmd.GenManTree("./")
	cmd.GenMarkdownTree("./")
}
