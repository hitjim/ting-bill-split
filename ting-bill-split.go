package main

import "fmt"
import "flag"
import "os"

func isBadParam(p *string) bool {
	return *p == ""
}

func main() {
	fmt.Println("Ting Bill Splitter\n")

	minPtr := flag.String("minutes", "", "filename for minutes csv")
	msgPtr := flag.String("messages", "", "filename for messages csv")
	megPtr := flag.String("megabytes", "", "filename for megabytes csv")

	flag.Parse()

	badParam := false

	if isBadParam(minPtr) {
		fmt.Println("minutes param is bad")
		badParam = true
	} else {
		fmt.Println("minutes:", *minPtr)
	}

	if isBadParam(msgPtr) {
		fmt.Println("messages param is bad")
		badParam = true
	} else {
		fmt.Println("messages:", *msgPtr)
	}

	if isBadParam(megPtr) {
		fmt.Println("megabytes param is bad")
		badParam = true
	} else {
		fmt.Println("megabytes:", *megPtr)
	}

	if badParam {
		os.Exit(1)
	}
}
