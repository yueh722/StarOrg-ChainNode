package commandlog

import (
	"chain/lib/cmdlogstore"
	"sort"
	"testing"
)

func TestCommandLog(t *testing.T) {
	var rs []cmdlogstore.Record

	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Time < rs[j].Time // 升序
	})

}
