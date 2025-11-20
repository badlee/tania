package app

import (
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/tools/osutils"
)

// the default pb_public dir location is relative to the executable
func defaultPublicDir() string {
	if osutils.IsProbablyGoRun() {
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}
