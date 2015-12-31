package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tv42/base58"
)

var prog = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s BASE58..\n", prog)
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix(prog + ": ")

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	for _, enc := range flag.Args() {
		buf := []byte(enc)
		dec, err := base58.DecodeToBig(buf)
		if err != nil {
			log.Fatal(err)
		}
		_, err = fmt.Println(dec)
		if err != nil {
			log.Fatal(err)
		}
	}
}
