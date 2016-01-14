package gcse

import (
	"github.com/daviddengcn/go-easybi"
)

func init() {
	bi.DataPath = BiDataPath.S()
}

func AddBiValueAndProcess(name string, value int) {
	bi.AddValue(name, value)
	bi.Flush()
	bi.Process()
}
