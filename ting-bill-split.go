package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

func checkParam(param string, ptr *string, badParam *bool) {
	if *ptr == "" {
		*badParam = true
		fmt.Printf("%s parameter is bad\n", param)
	}
}

func sliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}

// TODO remove after we handle all files
func readAndPrint(f *os.File) {
	data := make([]byte, 100)
	count, err := f.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("read %d bytes: %q\n", count, data[:count])
}

func parseMinutes(minReader io.Reader) map[string]int {
	var m map[string]int
	m = make(map[string]int)
	r := csv.NewReader(minReader)

	// Get index of important fields
	record, err := r.Read()

	if err == io.EOF {
		fmt.Println("minutes.csv is empty!")
		log.Fatal(err)
	} else if err != nil {
		fmt.Println("Error parsing minutes.csv")
		log.Fatal(err)
	}

	phoneIndex := sliceIndex(len(record), func(i int) bool { return record[i] == "Phone" })

	if phoneIndex < 0 {
		log.Fatal("Not a properly formed header on minutes.csv file!")
	}

	minIndex := sliceIndex(len(record), func(i int) bool { return record[i] == "Duration (min)" })

	if minIndex < 0 {
		log.Fatal("Not a properly formed header on minutes.csv file!")
	}

	for {
		record, err := r.Read()

		if err == io.EOF {
			break
		}

		min, err := strconv.Atoi(record[minIndex])
		if err != nil {
			log.Fatal(err)
		}

		phone := record[phoneIndex]

		if m[phone] > -1 {
			m[phone] += min
		} else {
			m[phone] = min
		}
	}

	return m
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

	readAndPrint(msgFile)
	readAndPrint(megFile)

	minMap := parseMinutes(bufio.NewReader(minFile))
	fmt.Println(minMap)
}
