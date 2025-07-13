package main

import (
	"log"

	"github.com/taigrr/teaqlite/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}