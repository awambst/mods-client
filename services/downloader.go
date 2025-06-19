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
	"regexp"
	"runtime"
	"strings"
	"time"

	"mod-installer/models"
)

type ProgressCallback func(downloaded, total int64)

type DownloadService struct {
	client    *http.Client
	tempDir   string
	cacheDir  string
	verifySum bool
}

func NewDownloadService(tempDir string, verifyChecksum bool) *DownloadService {
	ds := &DownloadService{
		client:    &http.Client{Timeout: 10 * time.Minute},
		tempDir:   tempDir,
		verifySum: verifyChecksum,
	}
	ds.cacheDir = ds.getCacheDir()
	ds.ensureDirectoryExists(ds.cacheDir)
	ds.ensureDirectoryExists(ds.tempDir)
	return ds
}

func (ds *DownloadService) getCacheDir() string {
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
			return filepath.Join(appData, "ModInstaller", "cache")
		}
	case "linux":
		if home := os.Getenv("HOME"); home != "" {
			return filepath.Join(home, ".cache", "mod-installer")
		}
	case "darwin":
		if home := os.Getenv("HOME"); home != "" {
			return filepath.Join(home, "Library", "Caches", "ModInstaller")
		}
	}
	return filepath.Join(os.TempDir(), "mod-installer-cache")
}

func (ds *DownloadService) ensureDirectoryExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func (ds *DownloadService) generateCacheKey(mod *models.Mod) string {
	hasher := sha256.New()
	hasher.Write([]byte(mod.DownloadURL))
	urlHash := fmt.Sprintf("%x", hasher.Sum(nil))[:8]
	safeName := strings.ReplaceAll(mod.ID, "/", "_")
	safeVersion := strings.ReplaceAll(mod.Version, "/", "_")
	return fmt.Sprintf("%s_%s_%s", safeName, safeVersion, urlHash)
}

func (ds *DownloadService) getCachedFilePath(mod *models.Mod) string {
	cacheKey := ds.generateCacheKey(mod)
	extension := ".zip"
	if strings.Contains(mod.DownloadURL, ".rar") {
		extension = ".rar"
	} else if strings.Contains(mod.DownloadURL, ".7z") {
		extension = ".7z"
	}
	return filepath.Join(ds.cacheDir, cacheKey+extension)
}

// NOUVEAU: Détection des dossiers Google Drive
func (ds *DownloadService) isGoogleDriveFolder(url string) bool {
	return strings.Contains(url, "drive.google.com/drive/folders/") ||
		   (strings.Contains(url, "drive.google.com") && strings.Contains(url, "folders"))
}

func (ds *DownloadService) isGoogleDriveURL(url string) bool {
	return strings.Contains(url, "drive.google.com") || strings.Contains(url, "docs.google.com")
}

func (ds *DownloadService) convertGoogleDriveURL(url string) string {
	patterns := []string{
		`/file/d/([a-zA-Z0-9_-]+)`,
		`id=([a-zA-Z0-9_-]+)`,
		`/d/([a-zA-Z0-9_-]+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			return fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", matches[1])
		}
	}
	return url
}

func (ds *DownloadService) IsModCached(mod *models.Mod) bool {
	cachedPath := ds.getCachedFilePath(mod)
	if info, err := os.Stat(cachedPath); err == nil && info.Size() > 1024 {
		if ds.verifySum && mod.Checksum != "" {
			if err := ds.verifyChecksum(cachedPath, mod.Checksum); err != nil {
				os.Remove(cachedPath)
				return false
			}
		}
		return true
	}
	return false
}

func (ds *DownloadService) GetCachedModPath(mod *models.Mod) string {
	if ds.IsModCached(mod) {
		return ds.getCachedFilePath(mod)
	}
	return ""
}

// MODIFIÉ: Ajout de la détection des dossiers
func (ds *DownloadService) DownloadMod(ctx context.Context, mod *models.Mod, callback ProgressCallback) (string, error) {
	// NOUVEAU: Vérifier si c'est un dossier Google Drive
	if ds.isGoogleDriveFolder(mod.DownloadURL) {
		return "", fmt.Errorf("dossier Google Drive détecté. Pour télécharger:\n1. Allez sur %s\n2. Sélectionnez tout (Ctrl+A)\n3. Clic droit > Télécharger\n4. Utilisez le ZIP créé", mod.DownloadURL)
	}

	cachedPath := ds.getCachedFilePath(mod)
	if ds.IsModCached(mod) {
		fmt.Printf("Mod %s trouvé en cache: %s\n", mod.ID, cachedPath)
		if callback != nil {
			callback(1, 1)
		}
		return cachedPath, nil
	}
	
	fmt.Printf("Téléchargement du mod %s...\n", mod.ID)
	
	tempFilename := fmt.Sprintf("download_%s_%d.tmp", ds.generateCacheKey(mod), time.Now().Unix())
	tempPath := filepath.Join(ds.tempDir, tempFilename)
	
	err := ds.downloadToFile(ctx, mod.DownloadURL, tempPath, callback)
	if err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("erreur téléchargement: %w", err)
	}
	
	if info, err := os.Stat(tempPath); err == nil && info.Size() < 1024 {
		os.Remove(tempPath)
		return "", fmt.Errorf("fichier trop petit (%d bytes)", info.Size())
	}
	
	if ds.verifySum && mod.Checksum != "" {
		if err := ds.verifyChecksum(tempPath, mod.Checksum); err != nil {
			os.Remove(tempPath)
			return "", fmt.Errorf("checksum invalide: %w", err)
		}
	}
	
	if err := os.Rename(tempPath, cachedPath); err != nil {
		os.Remove(tempPath)
		return "", fmt.Errorf("erreur mise en cache: %w", err)
	}
	
	fmt.Printf("Mod %s mis en cache: %s\n", mod.ID, cachedPath)
	return cachedPath, nil
}

func (ds *DownloadService) downloadToFile(ctx context.Context, url, filepath string, callback ProgressCallback) error {
	downloadURL := url
	if ds.isGoogleDriveURL(url) {
		downloadURL = ds.convertGoogleDriveURL(url)
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return err
	}
	
	resp, err := ds.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erreur HTTP: %s", resp.Status)
	}
	
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return fmt.Errorf("HTML reçu au lieu du fichier")
	}
	
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return ds.downloadWithProgress(resp.Body, file, resp.ContentLength, callback)
}

func (ds *DownloadService) downloadWithProgress(src io.Reader, dst io.Writer, total int64, callback ProgressCallback) error {
	var downloaded int64
	buf := make([]byte, 64*1024)
	
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
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
		return fmt.Errorf("checksum invalide")
	}
	return nil
}

func (ds *DownloadService) GetCacheSize() (int64, error) {
	var totalSize int64
	err := filepath.Walk(ds.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize, err
}

func (ds *DownloadService) ClearCache() error {
	return os.RemoveAll(ds.cacheDir)
}

func (ds *DownloadService) GetCacheInfo() (string, int64, int, error) {
	size, err := ds.GetCacheSize()
	if err != nil {
		return ds.cacheDir, 0, 0, err
	}
	
	fileCount := 0
	filepath.Walk(ds.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			fileCount++
		}
		return nil
	})
	
	return ds.cacheDir, size, fileCount, nil
}

func (ds *DownloadService) Cleanup() error {
	return os.RemoveAll(ds.tempDir)
}
