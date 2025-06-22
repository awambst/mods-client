// services/installer.go
package services

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mod-installer/models"
	"mod-installer/config"
	"mod-installer/utils"
	"mod-installer/utils/ntw"
)

// Utiliser le type de callback défini dans utils pour éviter l'import cyclique
type InstallProgressCallback = utils.InstallProgressCallback

// InstallerService gère l'installation des mods
type InstallerService struct {
	GamePath, ScriptsPath, TempDir, BackupDir string
	MakeBackups                               bool
}

// EnsureDirectoryExists crée un répertoire s'il n'existe pas
func (is *InstallerService) EnsureDirectoryExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func NewInstallerService(cfg *config.Config) *InstallerService {
	tempDir := cfg.TempPath
	backupDir := filepath.Join(tempDir, "backups")
	
	service := &InstallerService{
		GamePath:    cfg.GamePath,
		ScriptsPath: cfg.ScriptsPath,
		TempDir:     tempDir,
		BackupDir:   backupDir,
		MakeBackups: cfg.CreateBackups,
	}
	
	service.EnsureDirectoryExists(service.BackupDir)
	service.EnsureDirectoryExists(service.ScriptsPath)
	return service
}

func (is *InstallerService) GetDataPath() string {
	return filepath.Join(is.GamePath, "data")
}

func (is *InstallerService) GetScriptsPath() string {
	return is.ScriptsPath
}

func (is *InstallerService) IsGamePathValid() bool {
	if info, err := os.Stat(is.GetDataPath()); err == nil && info.IsDir() {
		return true
	}
	return false
}

func (is *InstallerService) InstallMod(ctx context.Context, mod *models.Mod, archivePath string, callback InstallProgressCallback) error {
	if !is.IsGamePathValid() {
		return fmt.Errorf("chemin du jeu invalide: %s", is.GamePath)
	}

	if is.MakeBackups {
		if err := is.createBackup(mod); err != nil {
			fmt.Printf("Attention: impossible de créer un backup: %v\n", err)
		}
	}

	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case ".zip":
		return is.extractZip(ctx, is.ScriptsPath, is.GamePath, archivePath, callback)
	case ".rar":
		return utils.ExtractRar(ctx, is.ScriptsPath, is.GamePath, archivePath, callback)
	case ".7z":
		return fmt.Errorf("format 7z non supporté dans cette version")
	default:
		return fmt.Errorf("format d'archive non supporté: %s", ext)
	}
}

func (is *InstallerService) extractZip(ctx context.Context, scriptsPath, gamePath, archivePath string, callback InstallProgressCallback) error {
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

		if err := is.extractFile(file.Name, destPath, file.FileInfo().IsDir(), file.FileInfo().Mode(), func() (io.ReadCloser, error) {
			return file.Open()
		}); err != nil {
			return fmt.Errorf("erreur extraction %s: %w", file.Name, err)
		}
	}
	return nil
}

func (is *InstallerService) extractFile(name, destPath string, isDir bool, mode os.FileMode, opener func() (io.ReadCloser, error)) error {
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

func (is *InstallerService) createBackup(mod *models.Mod) error {
	if !is.MakeBackups {
		return nil
	}

	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("backup_%s_%s_%s", mod.ID, mod.Version, timestamp)
	backupPath := filepath.Join(is.BackupDir, backupName)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("erreur création dossier backup: %w", err)
	}

	// Backup du dossier data
	dataBackupPath := filepath.Join(backupPath, "data")
	if err := is.copyDir(is.GetDataPath(), dataBackupPath); err != nil {
		fmt.Printf("Erreur backup data: %v\n", err)
	}

	// Backup du dossier scripts
	scriptsBackupPath := filepath.Join(backupPath, "scripts")
	if err := is.copyDir(is.GetScriptsPath(), scriptsBackupPath); err != nil {
		fmt.Printf("Erreur backup scripts: %v\n", err)
	}

	return nil
}

func (is *InstallerService) copyDir(src, dst string) error {
	// Vérifier si le dossier source existe
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil // Pas d'erreur si le dossier n'existe pas
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

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

func (is *InstallerService) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

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

func (is *InstallerService) GetBackups() ([]string, error) {
	entries, err := os.ReadDir(is.BackupDir)
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

func (is *InstallerService) RestoreBackup(backupName string) error {
	backupPath := filepath.Join(is.BackupDir, backupName)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup non trouvé: %s", backupName)
	}

	// Restaurer le dossier data
	dataBackupPath := filepath.Join(backupPath, "data")
	if _, err := os.Stat(dataBackupPath); err == nil {
		dataPath := is.GetDataPath()
		if err := os.RemoveAll(dataPath); err != nil {
			return fmt.Errorf("erreur suppression dossier data: %w", err)
		}
		if err := is.copyDir(dataBackupPath, dataPath); err != nil {
			return fmt.Errorf("erreur restauration data: %w", err)
		}
	}

	// Restaurer le dossier scripts
	scriptsBackupPath := filepath.Join(backupPath, "scripts")
	if _, err := os.Stat(scriptsBackupPath); err == nil {
		scriptsPath := is.GetScriptsPath()
		if err := os.RemoveAll(scriptsPath); err != nil {
			return fmt.Errorf("erreur suppression dossier scripts: %w", err)
		}
		if err := is.copyDir(scriptsBackupPath, scriptsPath); err != nil {
			return fmt.Errorf("erreur restauration scripts: %w", err)
		}
	}

	return nil
}

func (is *InstallerService) DeleteBackup(backupName string) error {
	return os.RemoveAll(filepath.Join(is.BackupDir, backupName))
}

func (is *InstallerService) GetInstallationStatus(mod *models.Mod) (bool, error) {
	return false, nil
}

func (is *InstallerService) Cleanup() error {
	matches, err := filepath.Glob(filepath.Join(is.TempDir, "TempDir", "install_*"))
	if err != nil {
		return err
	}

	for _, match := range matches {
		os.RemoveAll(match)
	}
	return nil
}
