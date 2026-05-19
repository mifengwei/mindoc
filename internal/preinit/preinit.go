// Package preinit performs os.Chdir to the executable's directory before
// other packages initialise, ensuring config files are found regardless
// of the working directory the binary is launched from.
package preinit

import (
	"os"
	"path/filepath"
	"strings"
)

func init() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exeDir := filepath.Dir(exe)
	if strings.Contains(exeDir, "go-build") {
		return
	}
	if _, err := os.Stat(filepath.Join(exeDir, "conf", "app.conf")); err != nil {
		return
	}
	_ = os.Chdir(exeDir)
}
