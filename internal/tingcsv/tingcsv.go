package tingcsv

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/hitjim/ting-bill-split/internal/tingbill"
	"github.com/shopspring/decimal"
)

// GenerateCSV accepts a tingbill.BillSplit, tingbill.Bill, filepath string, and returns a string
// containing a filepath for the newly generated Ting Bill Split CSV, and an error.
// The tingbill.Bill should be the same one that generated the tingbill.BillSplit
func GenerateCSV(bs tingbill.BillSplit, b tingbill.Bill, filePath string) (string, error) {
	fmt.Printf("\nGenerating invoice CSV...\n")
	const RoundPrecision = int32(2)

	// Bail right away if we can't write the CSV file
	csvFile, err := os.Create(filePath)
	if err != nil {
		return filePath, err
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Table 0: Heading - 7 columns, 2 rows
	// heading: Invoice filename w/date, Device qty, Bill Total, Split Total
	//   (for comparison), Usage subtotal, Devices Subtotal, Tax+Reg subtotal"

	// Prep data
	minCosts := decimal.New(0, 1)
	msgCosts := decimal.New(0, 1)
	megCosts := decimal.New(0, 1)
	shrCosts := decimal.New(0, 1)

	for _, v := range bs.MinuteCosts {
		minCosts = minCosts.Add(v)
	}

	for _, v := range bs.MessageCosts {
		msgCosts = msgCosts.Add(v)
	}
	for _, v := range bs.MegabyteCosts {
		megCosts = megCosts.Add(v)
	}
	for _, v := range bs.SharedCosts {
		shrCosts = shrCosts.Add(v)
	}

	calcCost := decimal.Sum(minCosts, msgCosts, megCosts, shrCosts).Round(RoundPrecision)
	usgCost := decimal.Sum(minCosts, msgCosts, megCosts).Round(RoundPrecision)

	records := [][]string{
		{"Invoice with date", "Devices Qty", "$Total", "$Calc", "$Usage", "$Devices", "$Tax+Reg"},
		{b.Description,
			strconv.Itoa(len(b.Devices)),
			strconv.FormatFloat(b.Total, 'f', 2, 64),
			calcCost.StringFixed(2),
			usgCost.StringFixed(2),
			strconv.FormatFloat(b.DevicesCost, 'f', 2, 64),
			strconv.FormatFloat(b.Fees, 'f', 2, 64)},
	}

	// Table 1: Usage - 8 columns, <deviceID qty>+1 rows
	// heading: number, nickname?, min, msg, data (KB), min%, msg%, data%
	// Then entries for each number
	// then entry for "Total" under nickname, and rest of sums

	// Table 2: Cost Type - 4 columns, 4 rows (+1 for cell to right of final column)
	// heading: Weighted Costs: Minutes, Messages, Data
	// Base: $x, $y, $z
	// Extra: etc
	// Total: etc (sum of Min, Msg, Data gets tacked on as extra cell/col on final row)

	// Table 3: Shared costs - 2 columns, 4 rows
	// heading: Type, Amount
	// Devices: $
	// Tax & Reg: $
	// Total: $

	// Table 4: Costs split - 7 columns, <deviceID qty>+1 rows
	// heading: number, Nickname, Min, Msg, Data, Shared, Total
	// entry for each number

	err = writer.WriteAll(records)

	return filePath, err
}
