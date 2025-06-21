// services/installer.go
package services

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"mod-installer/models"
)

type InstallProgressCallback func(currentFile string, processed, total int)

type InstallerService struct {
	gamePath    string
	tempDir     string
	backupDir   string
	makeBackups bool
}

func NewInstallerService(gamePath, tempDir string, makeBackups bool) *InstallerService {
	service := &InstallerService{
		gamePath:    gamePath,
		tempDir:     tempDir,
		makeBackups: makeBackups,
	}
	
	service.backupDir = filepath.Join(tempDir, "backups")
	service.ensureDirectoryExists(service.backupDir)
	
	return service
}

func (is *InstallerService) ensureDirectoryExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// GetDataPath retourne le chemin vers le dossier data du jeu
func (is *InstallerService) GetDataPath() string {
	return filepath.Join(is.gamePath, "data")
}

// IsGamePathValid vérifie si le chemin du jeu est valide
func (is *InstallerService) IsGamePathValid() bool {
	dataPath := is.GetDataPath()
	if info, err := os.Stat(dataPath); err == nil && info.IsDir() {
		return true
	}
	return false
}

// InstallMod installe un mod à partir d'un fichier téléchargé
func (is *InstallerService) InstallMod(ctx context.Context, mod *models.Mod, archivePath string, callback InstallProgressCallback) error {
	if !is.IsGamePathValid() {
		return fmt.Errorf("chemin du jeu invalide: %s", is.gamePath)
	}

	dataPath := is.GetDataPath()
	
	// Créer un backup si nécessaire
	if is.makeBackups {
		if err := is.createBackup(mod); err != nil {
			fmt.Printf("Attention: impossible de créer un backup: %v\n", err)
		}
	}

	// Déterminer le type d'archive et extraire
	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case ".zip":
		return is.extractZip(ctx, archivePath, dataPath, callback)
	case ".rar":
		return is.extractRar(ctx, archivePath, dataPath, callback)
	case ".7z":
		return is.extract7z(ctx, archivePath, dataPath, callback)
	default:
		return fmt.Errorf("format d'archive non supporté: %s", ext)
	}
}

// extractZip extrait une archive ZIP
func (is *InstallerService) extractZip(ctx context.Context, archivePath, destPath string, callback InstallProgressCallback) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("erreur ouverture ZIP: %w", err)
	}
	defer reader.Close()

	totalFiles := len(reader.File)
	processed := 0

	for _, file := range reader.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if callback != nil {
			callback(file.Name, processed, totalFiles)
		}

		if err := is.extractZipFile(file, destPath); err != nil {
			return fmt.Errorf("erreur extraction %s: %w", file.Name, err)
		}

		processed++
	}

	return nil
}

// extractZipFile extrait un fichier individuel d'une archive ZIP
func (is *InstallerService) extractZipFile(file *zip.File, destPath string) error {
	// Calculer le chemin de destination
	destFile := filepath.Join(destPath, file.Name)
	
	// Vérification de sécurité contre les path traversal
	if !strings.HasPrefix(destFile, filepath.Clean(destPath)+string(os.PathSeparator)) {
		return fmt.Errorf("chemin invalide: %s", file.Name)
	}

	// Créer les dossiers parents si nécessaire
	if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
		return err
	}

	// Si c'est un dossier, ne pas créer de fichier
	if file.FileInfo().IsDir() {
		return os.MkdirAll(destFile, file.FileInfo().Mode())
	}

	// Ouvrir le fichier dans l'archive
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	// Créer le fichier de destination
	outFile, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Copier le contenu
	_, err = io.Copy(outFile, rc)
	return err
}

// extractRar extrait une archive RAR en utilisant unrar
func (is *InstallerService) extractRar(ctx context.Context, archivePath, destPath string, callback InstallProgressCallback) error {
	// Vérifier si unrar est disponible
	if !is.isUnrarAvailable() {
		return fmt.Errorf("unrar n'est pas installé sur le système")
	}

	// Lister les fichiers dans l'archive pour le callback
	files, err := is.listRarFiles(archivePath)
	if err != nil {
		fmt.Printf("Attention: impossible de lister les fichiers RAR: %v\n", err)
		files = []string{"extraction en cours..."}
	}

	if callback != nil {
		callback("Préparation...", 0, len(files))
	}

	// Commande d'extraction
	cmd := exec.CommandContext(ctx, "unrar", "x", "-o+", archivePath, destPath+string(filepath.Separator))
	
	// Capturer la sortie pour le progress
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("erreur création pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erreur démarrage unrar: %w", err)
	}

	// Lire la sortie pour le progress
	go func() {
		buffer := make([]byte, 1024)
		processed := 0
		for {
			n, err := stdout.Read(buffer)
			if err != nil {
				break
			}
			output := string(buffer[:n])
			if strings.Contains(output, "Extracting") && callback != nil {
				processed++
				callback(fmt.Sprintf("Extraction... (%d/%d)", processed, len(files)), processed, len(files))
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("erreur unrar: %w", err)
	}

	if callback != nil {
		callback("Terminé", len(files), len(files))
	}

	return nil
}

// extract7z extrait une archive 7z
func (is *InstallerService) extract7z(ctx context.Context, archivePath, destPath string, callback InstallProgressCallback) error {
	// Vérifier si 7z est disponible
	if !is.is7zAvailable() {
		return fmt.Errorf("7zip n'est pas installé sur le système")
	}

	if callback != nil {
		callback("Extraction 7z...", 0, 1)
	}

	// Commande d'extraction
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "7z", "x", archivePath, "-o"+destPath, "-y")
	} else {
		cmd = exec.CommandContext(ctx, "7z", "x", archivePath, "-o"+destPath, "-y")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("erreur 7z: %w", err)
	}

	if callback != nil {
		callback("Terminé", 1, 1)
	}

	return nil
}

// isUnrarAvailable vérifie si unrar est disponible
func (is *InstallerService) isUnrarAvailable() bool {
	_, err := exec.LookPath("unrar")
	return err == nil
}

// is7zAvailable vérifie si 7z est disponible
func (is *InstallerService) is7zAvailable() bool {
	_, err := exec.LookPath("7z")
	if err != nil {
		// Essayer 7za sur Linux
		_, err = exec.LookPath("7za")
	}
	return err == nil
}

// listRarFiles liste les fichiers dans une archive RAR
func (is *InstallerService) listRarFiles(archivePath string) ([]string, error) {
	cmd := exec.Command("unrar", "lb", archivePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := strings.Split(string(output), "\n")
	var cleanFiles []string
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file != "" {
			cleanFiles = append(cleanFiles, file)
		}
	}

	return cleanFiles, nil
}

// createBackup crée un backup du dossier data
func (is *InstallerService) createBackup(mod *models.Mod) error {
	if !is.makeBackups {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("backup_%s_%s_%s", mod.ID, mod.Version, timestamp)
	backupPath := filepath.Join(is.backupDir, backupName)

	dataPath := is.GetDataPath()
	
	// Créer le dossier de backup
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("erreur création dossier backup: %w", err)
	}

	// Copier récursivement le dossier data
	return is.copyDir(dataPath, backupPath)
}

// copyDir copie récursivement un dossier
func (is *InstallerService) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculer le chemin de destination
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return is.copyFile(path, destPath)
	})
}

// copyFile copie un fichier
func (is *InstallerService) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Créer les dossiers parents si nécessaire
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// GetBackups retourne la liste des backups disponibles
func (is *InstallerService) GetBackups() ([]string, error) {
	entries, err := os.ReadDir(is.backupDir)
	if err != nil {
		return nil, err
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "backup_") {
			backups = append(backups, entry.Name())
		}
	}

	return backups, nil
}

// RestoreBackup restaure un backup
func (is *InstallerService) RestoreBackup(backupName string) error {
	backupPath := filepath.Join(is.backupDir, backupName)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup non trouvé: %s", backupName)
	}

	dataPath := is.GetDataPath()
	
	// Supprimer le dossier data actuel
	if err := os.RemoveAll(dataPath); err != nil {
		return fmt.Errorf("erreur suppression dossier data: %w", err)
	}

	// Restaurer depuis le backup
	return is.copyDir(backupPath, dataPath)
}

// DeleteBackup supprime un backup
func (is *InstallerService) DeleteBackup(backupName string) error {
	backupPath := filepath.Join(is.backupDir, backupName)
	return os.RemoveAll(backupPath)
}

// GetInstallationStatus vérifie si un mod est installé
func (is *InstallerService) GetInstallationStatus(mod *models.Mod) (bool, error) {
	// Cette fonction pourrait être améliorée en vérifiant des fichiers spécifiques
	// Pour l'instant, on considère qu'on ne peut pas détecter facilement l'installation
	return false, nil
}

// Cleanup nettoie les fichiers temporaires
func (is *InstallerService) Cleanup() error {
	// Nettoyer les fichiers temporaires mais pas les backups
	tempPattern := filepath.Join(is.tempDir, "install_*")
	matches, err := filepath.Glob(tempPattern)
	if err != nil {
		return err
	}

	for _, match := range matches {
		os.RemoveAll(match)
	}

	return nil
}
