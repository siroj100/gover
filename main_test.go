package gover

import (
	"os"
	"testing"
	"time"
)

var globalTimeLoc *time.Location

func TestMain(m *testing.M) {
	globalTimeLoc, _ = time.LoadLocation("Asia/Jakarta")

	os.Exit(m.Run())
}
