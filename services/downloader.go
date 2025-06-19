// services/downloader.go
package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"mod-installer/models"
)

// ProgressCallback est appelé pendant le téléchargement
type ProgressCallback func(downloaded, total int64)

// DownloadService gère les téléchargements de mods
type DownloadService struct {
	client    *http.Client
	tempDir   string
	verifySum bool
}

// NewDownloadService crée un nouveau service de téléchargement
func NewDownloadService(tempDir string, verifyChecksum bool) *DownloadService {
	return &DownloadService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		tempDir:   tempDir,
		verifySum: verifyChecksum,
	}
}

// DownloadMod télécharge un mod et vérifie son intégrité
func (ds *DownloadService) DownloadMod(ctx context.Context, mod *models.Mod, callback ProgressCallback) (string, error) {
	// Créer le nom de fichier temporaire
	filename := fmt.Sprintf("%s_%s.zip", mod.ID, mod.Version)
	filepath := filepath.Join(ds.tempDir, filename)
	
	// Créer la requête HTTP
	req, err := http.NewRequestWithContext(ctx, "GET", mod.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("erreur création requête: %w", err)
	}
	
	// Exécuter la requête
	resp, err := ds.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erreur téléchargement: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erreur HTTP: %s", resp.Status)
	}
	
	// Créer le fichier de destination
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("erreur création fichier: %w", err)
	}
	defer file.Close()
	
	// Télécharger avec callback de progression
	err = ds.downloadWithProgress(resp.Body, file, resp.ContentLength, callback)
	if err != nil {
		os.Remove(filepath) // Nettoyer en cas d'erreur
		return "", fmt.Errorf("erreur téléchargement: %w", err)
	}
	
	// Vérifier le checksum si demandé
	if ds.verifySum && mod.Checksum != "" {
		if err := ds.verifyChecksum(filepath, mod.Checksum); err != nil {
			os.Remove(filepath)
			return "", fmt.Errorf("erreur vérification: %w", err)
		}
	}
	
	return filepath, nil
}

// downloadWithProgress télécharge avec callback de progression
func (ds *DownloadService) downloadWithProgress(src io.Reader, dst io.Writer, total int64, callback ProgressCallback) error {
	var downloaded int64
	
	// Buffer pour la copie
	buf := make([]byte, 32*1024) // 32KB buffer
	
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			
			// Appeler le callback de progression
			if callback != nil {
				callback(downloaded, total)
			}
		}
		
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	
	return nil
}

// verifyChecksum vérifie l'intégrité du fichier téléchargé
func (ds *DownloadService) verifyChecksum(filepath, expectedChecksum string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	
	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum invalide: attendu %s, obtenu %s", expectedChecksum, actualChecksum)
	}
	
	return nil
}

// Cleanup nettoie les fichiers temporaires
func (ds *DownloadService) Cleanup() error {
	return os.RemoveAll(ds.tempDir)
}
