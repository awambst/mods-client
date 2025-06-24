package utils

import (
	"archive/zip"
	"context"
	"fmt"
	"io"

	"mod-installer/utils/ntw"
)


func ExtractZip(ctx context.Context, scriptsPath, gamePath, archivePath string, callback InstallProgressCallback) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture ZIP: %w", err)
	}
	defer reader.Close()

	for i, file := range reader.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if callback != nil {
			callback(file.Name, i, len(reader.File))
		}

		// Déterminer le dossier de destination basé sur l'extension
		destPath := ntw.GetDestinationPath(scriptsPath, gamePath, file.Name)

		if err := ExtractFile(file.Name, destPath, file.FileInfo().IsDir(), file.FileInfo().Mode(), func() (io.ReadCloser, error) {
			return file.Open()
		}); err != nil {
			return fmt.Errorf("erreur extraction %s: %w", file.Name, err)
		}
	}
	return nil
}
