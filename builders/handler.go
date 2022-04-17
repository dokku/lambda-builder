package builders

import (
	"fmt"
	"lambda-builder/io"
	"os"
	"path/filepath"
)

func getFunctionHandler(directory string, config Config) string {
	if config.Handler != "" {
		return config.Handler
	}

	for file, handler := range config.HandlerMap {
		if io.FileExistsInDirectory(directory, file) {
			return handler
		}
	}

	return ""
}

func writeProcfile(handler string, directory string) error {
	b := []byte(fmt.Sprintf("web: %s\n", handler))
	return os.WriteFile(filepath.Join(directory, "Procfile"), b, 0644)
}
