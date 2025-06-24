// utils/hash.go
package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// CalculateMD5 calcule le hash MD5 d'un fichier
func CalculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("impossible d'ouvrir le fichier %s: %v", filePath, err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("erreur lors du calcul MD5 pour %s: %v", filePath, err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// CalculateSHA256 calcule le hash SHA256 d'un fichier (plus sécurisé)
func CalculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("impossible d'ouvrir le fichier %s: %v", filePath, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("erreur lors du calcul SHA256 pour %s: %v", filePath, err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// VerifyFileIntegrity vérifie l'intégrité d'un fichier avec son checksum
func VerifyFileIntegrity(filePath, expectedChecksum string) (bool, error) {
	currentChecksum, err := CalculateMD5(filePath)
	if err != nil {
		return false, err
	}
	
	return currentChecksum == expectedChecksum, nil
}

// GenerateFileID génère un ID unique pour un fichier basé sur son chemin et sa taille
func GenerateFileID(filePath string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	
	// Utilise le chemin et la taille pour créer un ID unique
	data := fmt.Sprintf("%s_%d", filePath, info.Size())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash), nil
}
