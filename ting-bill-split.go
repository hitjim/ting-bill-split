package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func isBadParam(p *string) bool {
	return *p == ""
}

func checkParam() func(string, *string, *bool) {
	return func(param string, ptr *string, badParam *bool) {
		if isBadParam(ptr) {
			fmt.Printf("%s parameter is bad\n", param)
		} else {
			fmt.Printf("%s: ", param)
			fmt.Printf(*ptr)
			fmt.Printf("\n")
		}
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
	bParamPtr := &badParam
	paramMap := map[string]*string{
		"minutes":   minPtr,
		"messages":  msgPtr,
		"megabytes": megPtr,
	}

	check := checkParam()

	for k, v := range paramMap {
		check(k, v, bParamPtr)
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
