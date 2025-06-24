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

// CopyFile copie un fichier de src vers dst
func CopyFile(src, dst string) error {
	// Créer le répertoire de destination s'il n'existe pas
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("impossible de créer le répertoire de destination: %v", err)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir le fichier source %s: %v", src, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier de destination %s: %v", dst, err)
	}
	defer destFile.Close()

	// Copier le contenu
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("erreur lors de la copie: %v", err)
	}

	// Copier les permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("impossible de lire les informations du fichier source: %v", err)
	}

	if err := destFile.Chmod(sourceInfo.Mode()); err != nil {
		return fmt.Errorf("impossible de définir les permissions: %v", err)
	}

	return nil
}

// FileExists vérifie si un fichier existe
func FileExists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// GetFileSize retourne la taille d'un fichier en octets
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// EnsureDirectoryExists crée un répertoire s'il n'existe pas
func EnsureDirectoryExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// SafeFileMove déplace un fichier de manière sécurisée (copie puis supprime l'original)
func SafeFileMove(src, dst string) error {
	// D'abord copier le fichier
	if err := CopyFile(src, dst); err != nil {
		return fmt.Errorf("échec de la copie lors du déplacement: %v", err)
	}

	// Puis supprimer l'original
	if err := os.Remove(src); err != nil {
		// Si la suppression échoue, essayer de supprimer la copie
		os.Remove(dst)
		return fmt.Errorf("échec de la suppression du fichier source: %v", err)
	}

	return nil
}

// GetRelativePath retourne le chemin relatif d'un fichier par rapport à un répertoire de base
func GetRelativePath(basePath, filePath string) (string, error) {
	rel, err := filepath.Rel(basePath, filePath)
	if err != nil {
		return "", fmt.Errorf("impossible de calculer le chemin relatif: %v", err)
	}
	return rel, nil
}
