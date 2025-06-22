package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractFile(name, destPath string, isDir bool, mode os.FileMode, opener func() (io.ReadCloser, error)) error {
	destFile := filepath.Join(destPath, name)
	
	// Vérification de sécurité contre les path traversal
	if !strings.HasPrefix(destFile, filepath.Clean(destPath)+string(os.PathSeparator)) {
		return fmt.Errorf("chemin invalide: %s", name)
	}

	if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
		return err
	}

	if isDir {
		return os.MkdirAll(destFile, mode)
	}

	rc, err := opener()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}
