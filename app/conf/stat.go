package conf

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type stat struct {
	total atomic.Int64
	fail  atomic.Int64
}

var (
	data sync.Map // map[string]*stat
)

func getStat(net string) *stat {
	val, _ := data.LoadOrStore(net, &stat{})

	return val.(*stat)
}

func SetBlockTotal(net string) {
	getStat(net).total.Add(1)
}

func SetBlockFail(net string) {
	getStat(net).fail.Add(1)
}

func GetBlockSuccRate(net string) string {
	s := getStat(net)
	t := s.total.Load()
	if t == 0 {

		return "100.00%"
	}

	f := s.fail.Load()

	return fmt.Sprintf("%.2f%%", float64(t-f)/float64(t)*100)
}
