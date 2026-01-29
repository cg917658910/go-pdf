package main

import (
	"log"
	"time"

	eng "pdfguard/new"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func main() {
	in := "m.pdf"
	out := "m_protected.pdf"

	ctx, err := api.ReadContextFile(in)
	if err != nil {
		log.Fatalf("read context: %v", err)
	}

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 23, 59, 59, 0, time.UTC)

	if err := eng.NewEngineRun(ctx, start, end, "Unsupported viewer: please use Acrobat/Reader with JS enabled"); err != nil {
		log.Fatalf("protect: %v", err)
	}

	if err := api.WriteContextFile(ctx, out); err != nil {
		log.Fatalf("write: %v", err)
	}
	log.Printf("wrote %s", out)
}
