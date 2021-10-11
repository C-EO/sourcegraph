package policies

import (
	"time"

	"github.com/sourcegraph/sourcegraph/internal/timeutil"
)

// Test repository:
//
//                                              v2.2.2                                        feat/blank
//                                             /                                             /
//  09               08 ---- 07              06              05 ------ 04 ------ 03 ------ 02 ------ 01
//   \                        \               \               \         \                             \
//    ef/feature-y            ef/feature-x    es/feature-z     v1.2.2    v1.2.3                        develop

var testBranchHeads = map[string]string{
	"develop":      "deadbeef01",
	"feat/blank":   "deadbeef02",
	"ef/feature-x": "deadbeef07",
	"es/feature-z": "deadbeef06",
	"ef/feature-y": "deadbeef09",
}

var testTagHeads = map[string]string{
	"v1.2.3": "deadbeef04",
	"v1.2.2": "deadbeef05",
	"v2.2.2": "deadbeef06",
}

var testBranchMembers = map[string][]string{
	"develop":      {"deadbeef01", "deadbeef02", "deadbeef03", "deadbeef04", "deadbeef05"},
	"feat/blank":   {"deadbeef02"},
	"ef/feature-x": {"deadbeef07", "deadbeef08"},
	"ef/feature-y": {"deadbeef09"},
	"es/feature-z": {"deadbeef06"},
}

var testNow = timeutil.Now()

var testCreatedAt = map[string]time.Time{
	"deadbeef01": testNow.Add(-time.Hour * 5),
	"deadbeef02": testNow.Add(-time.Hour * 5),
	"deadbeef03": testNow.Add(-time.Hour * 5),
	"deadbeef07": testNow.Add(-time.Hour * 5),
	"deadbeef08": testNow.Add(-time.Hour * 5),
	"deadbeef04": testNow.Add(-time.Hour * 5),
	"deadbeef05": testNow.Add(-time.Hour * 12),
	"deadbeef06": testNow.Add(-time.Hour * 15),
	"deadbeef09": testNow.Add(-time.Hour * 15),
}
