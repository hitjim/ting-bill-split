package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func checkParam(param string, ptr *string, badParam *bool) {
	if *ptr == "" {
		*badParam = true
		fmt.Printf("%s parameter is bad\n", param)
	}
}

func readAndPrint(f *os.File) {
	data := make([]byte, 100)
	count, err := f.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("read %d bytes: %q\n", count, data[:count])
}

func main() {
	fmt.Printf("Ting Bill Splitter\n\n")

	minPtr := flag.String("minutes", "", "filename for minutes csv")
	msgPtr := flag.String("messages", "", "filename for messages csv")
	megPtr := flag.String("megabytes", "", "filename for megabytes csv")

	flag.Parse()

	badParam := false
	paramMap := map[string]*string{
		"minutes":   minPtr,
		"messages":  msgPtr,
		"megabytes": megPtr,
	}

	for k, v := range paramMap {
		checkParam(k, v, &badParam)
	}

	if badParam {
		os.Exit(1)
	}

	minFile, err := os.Open(*minPtr)
	if err != nil {
		log.Fatal(err)
	}

	msgFile, err := os.Open(*msgPtr)
	if err != nil {
		log.Fatal(err)
	}

	megFile, err := os.Open(*megPtr)
	if err != nil {
		log.Fatal(err)
	}

	readAndPrint(minFile)
	readAndPrint(msgFile)
	readAndPrint(megFile)
}
