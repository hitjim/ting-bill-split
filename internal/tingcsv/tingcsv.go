package tingcsv

import (
	"fmt"

	"github.com/hitjim/ting-bill-split/internal/tingbill"
)

// GenerateCSV accepts a tingbill.BillSplit, tingbill.Bill, filepath string, and returns a string
// containing a filepath for the newly generated Ting Bill Split CSV, and an error.
// The tingbill.Bill should be the same one that generated the tingbill.BillSplit
func GenerateCSV(bs tingbill.BillSplit, b tingbill.Bill, filePath string) (string, error) {
	fmt.Printf("\nGenerating invoice CSV...\n")
	RoundPrecision := int32(2)

	// Table 0: Heading - 7 rows
	// heading: Invoice filename w/date, Device qty, Bill Total, Split Total
	//   (for comparison), Usage subtotal, Devices Subtotal, Tax+Reg subtotal"

	// Table 1: Usage - 8 rows
	// heading: number, nickname?, min, msg, data (KB), min%, msg%, data%
	// Then entries for each number
	// then entry for "Total" under nickname, and rest of sums

	// Table 2: Weighted costs - 3 rows
	// heading: Weighted Costs: Minutes, Messages, Data
	// Base: $x, $y, $z
	// Extra: etc
	// Total: etc

	// Table 3: Shared costs - 2
	// TODO LATER - handle all the tax and reg costs in bill file?
	// heading: Type, Amount

	// Table 4: Costs split - 7
	// heading: number, Nickname, Min, Msg, Data, Shared, Total
	// entry for each number

	return filePath, err
}
