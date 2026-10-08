package grpc

import (
	"os"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
)

func TestMain(m *testing.M) { os.Exit(testsupport.Main(m)) }
