package adapters

import (
	"os"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
)

// TestMain tears the shared container down after the package's tests.
func TestMain(m *testing.M) { os.Exit(testsupport.Main(m)) }
