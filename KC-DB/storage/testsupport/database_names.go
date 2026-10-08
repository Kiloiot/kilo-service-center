package testsupport

import (
	"fmt"
	"os"
	"strconv"
)

// processTag keeps one test binary's databases apart from another's: `go test
// ./...` runs packages as parallel processes, and against an external server
// (TEST_DB_HOST) they share one catalog.
var processTag = strconv.Itoa(os.Getpid())

// templateName is this process's migrated template database.
func templateName() string {
	return templateDatabase + "_" + processTag
}

// testDatabaseName names this process's seq-th per-test database.
func testDatabaseName(seq uint64) string {
	return fmt.Sprintf("%s%s_%d", testDatabasePrefix, processTag, seq)
}
