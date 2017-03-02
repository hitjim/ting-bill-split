package main

import "fmt"
import "flag"
import "os"
import "log"

func isBadParam(p *string) bool {
	return *p == ""
}

func checkParams(minp *string, msgp *string, megp *string) {
	badParam := false

	if isBadParam(minp) {
		fmt.Println("minutes param is bad")
		badParam = true
	} else {
		fmt.Println("minutes:", *minp)
	}

	if isBadParam(msgp) {
		fmt.Println("messages param is bad")
		badParam = true
	} else {
		fmt.Println("messages:", *msgp)
	}

	if isBadParam(megp) {
		fmt.Println("megabytes param is bad")
		badParam = true
	} else {
		fmt.Println("megabytes:", *megp)
	}

	if badParam {
		os.Exit(1)
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
	fmt.Println("Ting Bill Splitter\n")

	minPtr := flag.String("minutes", "", "filename for minutes csv")
	msgPtr := flag.String("messages", "", "filename for messages csv")
	megPtr := flag.String("megabytes", "", "filename for megabytes csv")

	flag.Parse()

	checkParams(minPtr, msgPtr, megPtr)

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
