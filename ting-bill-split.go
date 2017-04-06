package main

import (
	"encoding/csv"
	"errors"
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

func parseMinutes(minReader io.Reader) (map[string]int, error) {
	m := make(map[string]int)
	r := csv.NewReader(minReader)

	// Get index of important fields
	header, err := r.Read()

	if err == io.EOF {
		fmt.Println("minutes.csv is empty!")
		return m, err
	}

	if err != nil {
		fmt.Println("Error parsing minutes.csv")
		return m, err
	}

	phoneIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Phone" })

	if phoneIndex < 0 {
		return m, errors.New(`missing "Phone" header in minutes csv file`)
	}

	minIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Duration (min)" })

	if minIndex < 0 {
		return m, errors.New(`missing "Duration (min)" header in minutes csv file`)
	}

	for {
		record, err := r.Read()

		if err != nil {
			if err == io.EOF {
				break
			}
			return m, err
		}

		min, err := strconv.Atoi(record[minIndex])
		if err != nil {
			return m, err
		}

		phone := record[phoneIndex]

		m[phone] += min
	}

	return m, nil
}

func parseMessages(msgReader io.Reader) (map[string]int, error) {
	m := make(map[string]int)
	r := csv.NewReader(msgReader)

	// Get index of important fields
	header, err := r.Read()

	if err == io.EOF {
		fmt.Println("messages.csv is empty!")
		return m, err
	}

	if err != nil {
		fmt.Println("Error parsing messages.csv")
		return m, err
	}

	phoneIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Phone" })

	if phoneIndex < 0 {
		return m, errors.New(`missing "Phone" header in messages csv file`)
	}

	for {
		record, err := r.Read()

		if err != nil {
			if err == io.EOF {
				break
			}
			return m, err
		}

		phone := record[phoneIndex]

		m[phone]++
	}

	return m, nil
}

func parseMegabytes(megReader io.Reader) (map[string]int, error) {
	m := make(map[string]int)
	r := csv.NewReader(megReader)

	// Get index of important fields
	header, err := r.Read()

	if err == io.EOF {
		fmt.Println("megabytes.csv is empty!")
		return m, err
	}

	if err != nil {
		fmt.Println("Error parsing megabytes.csv")
		return m, err
	}

	phoneIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Device" })

	if phoneIndex < 0 {
		return m, errors.New(`missing "Device" header in megabytes csv file`)
	}

	kbIndex := sliceIndex(len(header), func(i int) bool { return header[i] == "Kilobytes" })

	if kbIndex < 0 {
		return m, errors.New(`missing "Kilobytes" header in megabytes csv file`)
	}

	for {
		record, err := r.Read()

		if err != nil {
			if err == io.EOF {
				break
			}
			return m, err
		}

		kb, err := strconv.Atoi(record[kbIndex])
		if err != nil {
			return m, err
		}

		phone := record[phoneIndex]

		m[phone] += kb
	}

	return m, nil
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

	minMap, err := parseMinutes(minFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(minMap)

	msgMap, err := parseMessages(msgFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(msgMap)

	megMap, err := parseMegabytes(megFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(megMap)
}
