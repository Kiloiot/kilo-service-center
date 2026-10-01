package bssci

import (
	"os"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
)

// TestMain tears the shared database container down after the package's tests.
func TestMain(m *testing.M) { os.Exit(testsupport.Main(m)) }
