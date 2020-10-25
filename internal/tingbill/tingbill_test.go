package tingbill

import (
	"fmt"
	"testing"
)

func TestBillOwnerByID(t *testing.T) {
	b := Bill{}

	t.Run("Handle empty id", testBillOwnerByIDFunc(b, "", "Unknown"))
}

func testBillOwnerByIDFunc(b Bill, id string, expected string) func(*testing.T) {
	return func(t *testing.T) {
		actual := b.OwnerByID(id)
		if actual != expected {
			t.Error(fmt.Sprintf("Expected owner of %v to be %s but instead got %s", id, expected, actual))
		}
	}
}
