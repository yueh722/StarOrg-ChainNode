package commandlog

import (
	"Chain/lib/logdb"
	"sort"
	"testing"
)

func TestCommandLog(t *testing.T) {
	var rs []logdb.Record

	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Time < rs[j].Time // 升序
	})

}
