package tingparse

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/hitjim/ting-bill-split/internal/tingbill"
	"github.com/shopspring/decimal"
)

func sliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}

func ParseBill(r io.Reader) (tingbill.Bill, error) {
	var b tingbill.Bill
	if _, err := toml.DecodeReader(r, &b); err != nil {
		return tingbill.Bill{}, err
	}

	ids := b.DeviceIds()

	// Check to see if a shortStrawId was set. If not, set it to first one we find.
	// Ordering is random, so deal with it.
	// TODO - make this more testable
	phoneIndex := sliceIndex(len(ids), func(i int) bool { return ids[i] == b.ShortStrawID })

	if phoneIndex < 0 {
		b.ShortStrawID = ids[0]
	}

	return b, nil
}

func ParseMinutes(minReader io.Reader) (map[string]int, error) {
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

func ParseMessages(msgReader io.Reader) (map[string]int, error) {
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

func ParseMegabytes(megReader io.Reader) (map[string]int, error) {
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

func ParseMaps(min map[string]int, msg map[string]int, meg map[string]int, bil tingbill.Bill) (tingbill.BillSplit, error) {
	bs := tingbill.BillSplit{
		make(map[string]decimal.Decimal),
		make(map[string]int),
		make(map[string]decimal.Decimal),
		make(map[string]decimal.Decimal),
		make(map[string]int),
		make(map[string]decimal.Decimal),
		make(map[string]decimal.Decimal),
		make(map[string]int),
		make(map[string]decimal.Decimal),
		make(map[string]decimal.Decimal),
	}
	var usedMin, usedMsg, usedMeg int
	DecimalPrecision := int32(6)

	bilMinutes := decimal.NewFromFloat(bil.Minutes + bil.ExtraMinutes)
	bilMessages := decimal.NewFromFloat(bil.Messages + bil.ExtraMessages)
	bilMegabytes := decimal.NewFromFloat(bil.Megabytes + bil.ExtraMegabytes)
	delta := decimal.NewFromFloat(bil.DevicesCost + bil.Fees).Round(DecimalPrecision)
	deviceQty := decimal.New(int64(len(bil.Devices)), 0)

	// Calculate usage totals
	for _, v := range min {
		usedMin += v
	}

	for _, v := range msg {
		usedMsg += v
	}

	for _, v := range meg {
		usedMeg += v
	}

	totalMin := decimal.New(int64(usedMin), DecimalPrecision)
	totalMsg := decimal.New(int64(usedMsg), DecimalPrecision)
	totalMeg := decimal.New(int64(usedMeg), DecimalPrecision)

	deviceIds := bil.DeviceIds()

	for _, id := range deviceIds {
		subMin := decimal.New(int64(min[id]), DecimalPrecision)
		bs.MinutePercent[id] = subMin.Div(totalMin)
		bs.MinuteCosts[id] = bs.MinutePercent[id].Mul(bilMinutes)
		// It's possible for a device to still be on the Bill, but not show any usage data
		if value, exists := min[id]; exists {
			bs.MinuteQty[id] = value
		} else {
			bs.MinuteQty[id] = 0
		}

		subMsg := decimal.New(int64(msg[id]), DecimalPrecision)
		bs.MessagePercent[id] = subMsg.DivRound(totalMsg, DecimalPrecision)
		bs.MessageCosts[id] = bs.MessagePercent[id].Mul(bilMessages)
		// It's possible for a device to still be on the Bill, but not show any usage data
		if value, exists := msg[id]; exists {
			bs.MessageQty[id] = value
		} else {
			bs.MessageQty[id] = 0
		}

		subMeg := decimal.New(int64(meg[id]), DecimalPrecision)
		bs.MegabytePercent[id] = subMeg.DivRound(totalMeg, DecimalPrecision)
		bs.MegabyteCosts[id] = bs.MegabytePercent[id].Mul(bilMegabytes)
		// It's possible for a device to still be on the Bill, but not show any usage data
		if value, exists := meg[id]; exists {
			bs.MegabyteQty[id] = value
		} else {
			bs.MegabyteQty[id] = 0
		}

		bs.SharedCosts[id] = delta.DivRound(deviceQty, DecimalPrecision)
	}

	minSubSum := decimal.New(0, DecimalPrecision)
	for _, sub := range bs.MinuteCosts {
		minSubSum = minSubSum.Add(sub)
	}

	minSubExtra := bilMinutes.Sub(minSubSum)
	if minSubExtra.GreaterThan(decimal.New(0, DecimalPrecision)) {
		fmt.Printf("Remainder minutes cost of $%s to deviceId %s\n", minSubExtra.String(), bil.ShortStrawID)
		bs.MinuteCosts[bil.ShortStrawID].Add(minSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting minutes.")
	}

	msgSubSum := decimal.New(0, DecimalPrecision)
	for _, sub := range bs.MessageCosts {
		msgSubSum = msgSubSum.Add(sub)
	}

	msgSubExtra := bilMessages.Sub(msgSubSum)
	if msgSubExtra.GreaterThan(decimal.New(0, DecimalPrecision)) {
		fmt.Printf("Remainder messages cost of $%s to deviceId %s\n", msgSubExtra.String(), bil.ShortStrawID)
		bs.MessageCosts[bil.ShortStrawID].Add(msgSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting messages.")
	}

	megSubSum := decimal.New(0, DecimalPrecision)
	for _, sub := range bs.MegabyteCosts {
		megSubSum = megSubSum.Add(sub)
	}

	megSubExtra := bilMessages.Sub(megSubSum)
	if megSubExtra.GreaterThan(decimal.New(0, DecimalPrecision)) {
		fmt.Printf("Remainder megabytes cost of $%s to deviceId %s\n", megSubExtra.String(), bil.ShortStrawID)
		bs.MegabyteCosts[bil.ShortStrawID].Add(megSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting megabytes.")
	}

	deltaSubSum := decimal.New(0, DecimalPrecision)
	for _, sub := range bs.SharedCosts {
		deltaSubSum = deltaSubSum.Add(sub)
	}

	deltaSubExtra := delta.Sub(deltaSubSum)
	if deltaSubExtra.GreaterThan(decimal.New(0, DecimalPrecision)) {
		fmt.Printf("Remainder delta cost of $%s added to deviceId %s\n", deltaSubExtra.String(), bil.ShortStrawID)
		bs.SharedCosts[bil.ShortStrawID] = bs.SharedCosts[bil.ShortStrawID].Add(deltaSubExtra)
	} else {
		fmt.Println("There was no remainder cost when splitting delta.")
	}

	return bs, nil
}
