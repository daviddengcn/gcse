package gcse

import (
	"github.com/daviddengcn/go-easybi"
)

func init() {
	bi.DataPath = BiDataPath.S()
}

func AddBiValueAndProcess(aggr bi.AggregateMethod, name string, value int) {
	bi.AddValue(aggr, name, value)
	bi.Flush()
	bi.Process()
}
