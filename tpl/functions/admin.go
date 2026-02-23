package functions

import (
	"fmt"

	"github.com/ignisVeneficus/lumenta/data"
)

func FocusOffset(f data.Focus) string {
	return fmt.Sprintf("--fx: %.2f; --fy: %.2f;", f.FocusX, f.FocusY)
}
