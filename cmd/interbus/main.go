package main

import (
	"log"

	"solidcoredata.org/src/databus/inter"
)

func main() {
	err := inter.RunCLI()
	if err != nil {
		log.Fatal(err)
	}
}
