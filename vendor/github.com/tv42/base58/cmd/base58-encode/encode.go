package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"github.com/tv42/base58"
)

var prog = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s NUMBER..\n", prog)
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

	for _, dec := range flag.Args() {
		num := new(big.Int)
		if _, ok := num.SetString(dec, 10); !ok {
			log.Fatalf("not a number: %s", dec)
		}

		buf := base58.EncodeBig(nil, num)
		if _, err := fmt.Printf("%s\n", buf); err != nil {
			log.Fatal(err)
		}
	}
}
