package main

import (
	"log"
	"time"

	eng "pdfguard/internal/engine/v2"
)

func main() {
	in := "./cg.pdf"
	out := "./m2_protected.pdf"

	/* ctx, err := api.ReadContextFile(in)
	if err != nil {
		log.Fatalf("read context: %v", err)
	} */

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 1, 23, 59, 59, 0, time.UTC)
	opts := eng.Options{
		Input:     in,
		Output:    out,
		StartTime: start,
		EndTime:   end,
		Watermark: "Unspport view",
	}
	if err := eng.Run(opts); err != nil {
		log.Fatalf("protect: %v", err)
	}

	log.Printf("wrote %s", out)
}
