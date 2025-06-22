package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nwaples/rardecode/v2"

	"mod-installer/utils/ntw"
)

// InstallProgressCallback définit le type de callback pour le progrès d'installation
// Déplacé ici pour éviter l'import cyclique
type InstallProgressCallback func(currentFile string, processed, total int)

func ExtractRar(ctx context.Context, scriptsPath, gamePath, archivePath string, callback InstallProgressCallback) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture RAR: %w", err)
	}
	defer file.Close()

	reader, err := rardecode.NewReader(file)
	if err != nil {
		return fmt.Errorf("erreur création lecteur RAR: %w", err)
	}

	processed := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("erreur lecture header RAR: %w", err)
		}

		if callback != nil {
			callback(header.Name, processed, 0)
		}

		// Déterminer le dossier de destination basé sur l'extension
		destPath := ntw.GetDestinationPath(scriptsPath, gamePath, header.Name)

		if err := ExtractFile(header.Name, destPath, header.IsDir, 0644, func() (io.ReadCloser, error) {
			return io.NopCloser(reader), nil
		}); err != nil {
			return fmt.Errorf("erreur extraction %s: %w", header.Name, err)
		}

		if !header.ModificationTime.IsZero() {
			destFile := filepath.Join(destPath, header.Name)
			os.Chtimes(destFile, header.ModificationTime, header.ModificationTime)
		}
		processed++
	}
	return nil
}
