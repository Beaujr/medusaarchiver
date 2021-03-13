package main

import (
	"flag"
	"github.com/beaujr/medusaarchiver/medusa"
	"log"
)

func main() {
	flag.Parse()
	err := medusa.StartUpdate()
	if err != nil {
		log.Fatal(err)
	}
}
