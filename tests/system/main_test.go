package system

import (
	"os"
	"testing"

	"github.com/inframind/inframind/tests/internal/harness"
)

func TestMain(m *testing.M) {
	code := m.Run()
	harness.CloseGlobal()
	os.Exit(code)
}
