package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/pellux-network/EDxDC/edreader"
)

func main() {
	filename := os.Args[1]

	f, err := os.Open(filename)
	if err != nil {
		log.Panicln(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	state := edreader.Journalstate{}
	for scanner.Scan() {
		line := scanner.Bytes()
		edreader.ParseJournalLine(line, &state)
		fmt.Printf("%#v\n", state)
	}

}
