package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cryptonstudio/crypton-matching-engine/matching"
	"github.com/cryptonstudio/crypton-matching-engine/providers/nasdaq/itch"
)

const filePath = "./.stash/itch/01302019.NASDAQ_ITCH50"

const multithread = true
const autoMatching = true

var _ itch.Handler = &ITCH{}
var _ matching.Handler = &Matcher{}

func main() {

	// Create matching engine
	handler := &Matcher{}
	engine := matching.NewEngine(handler, multithread)

	// Disable auto matching
	if autoMatching && !engine.IsMatchingEnabled() {
		engine.EnableMatching()
	}

	// Create ITCH data processor
	itchHandler := &ITCH{engine: engine}
	processor, err := itch.NewProcessor(itchHandler)
	if err != nil {
		log.Fatal(err)
	}

	// Start matching engine
	timeStart := time.Now()
	engine.Start()

	// Run reading ITCH data from file
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	err = processor.Process(file)
	if err != nil {
		log.Fatal(err)
	}

	// Stop matching engine
	engine.Stop(false)
	timeElapsed := time.Since(timeStart)

	// Print statistics
	fmt.Println()
	itchHandler.PrintStatistics(timeElapsed)
	fmt.Println()
	handler.PrintStatistics(timeElapsed)
	fmt.Println()
	fmt.Printf("Time elapsed: %f seconds\n", timeElapsed.Seconds())
}
