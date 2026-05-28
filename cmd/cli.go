package main

import (
	"fmt"
	kafcli "github.com/ystyle/kaf-cli/internal/kafcli"
	"os"
	"strings"
)

var (
	secret      string
	measurement string
	version     string
)

func main() {
	var book *kafcli.Book
	var err error
	if len(os.Args) == 2 && strings.HasSuffix(os.Args[1], ".txt") {
		book, err = kafcli.NewBookSimple(os.Args[1])
		if err != nil {
			fmt.Printf("错误: %s\n", err.Error())
			os.Exit(3)
		}
	} else {
		book = kafcli.NewBookArgs()
	}
	if err := kafcli.Run(book, version, secret, measurement); err != nil {
		fmt.Printf("错误: %s\n", err.Error())
		os.Exit(1)
	}
}
