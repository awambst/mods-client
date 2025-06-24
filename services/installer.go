// services/installer.go
package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mod-installer/models"
	"mod-installer/config"
	"mod-installer/utils"
)

// Utiliser le type de callback défini dans utils pour éviter l'import cyclique
type InstallProgressCallback = utils.InstallProgressCallback

// InstallerService gère l'installation des mods
type InstallerService struct {
	gamePath, scriptsPath, TempDir string
}

// EnsureDirectoryExists crée un répertoire s'il n'existe pas
func (is *InstallerService) EnsureDirectoryExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func NewInstallerService(cfg *config.Config) *InstallerService {
	service := &InstallerService{
		gamePath:    cfg.GamePath,
		scriptsPath: cfg.ScriptsPath,
		TempDir:     cfg.TempPath,
	}

	
	service.EnsureDirectoryExists(service.GetScriptsPath())
	return service
}

func (is *InstallerService) GetDataPath() string {
	return filepath.Join(is.gamePath, "data")
}

func (is *InstallerService) GetScriptsPath() string {
	return filepath.Join(is.scriptsPath, "scripts")
}

func (is *InstallerService) IsGamePathValid() bool {

	if info, err := os.Stat(is.GetDataPath()); err == nil && info.IsDir() {
		return strings.HasSuffix(is.gamePath, "Napoleon Total War") 
	}
	return false
}

func (is *InstallerService) IsScriptsPathValid() bool {

	if info, err := os.Stat(is.GetScriptsPath()); err == nil && info.IsDir() {
		return strings.HasSuffix(is.scriptsPath, "Napoleon") 
	}
	return false
}

func (is *InstallerService) InstallMod(ctx context.Context, mod *models.Mod, archivePath string, callback InstallProgressCallback) error {
	if !is.IsGamePathValid() {
		return fmt.Errorf("chemin du jeu invalide: %s", is.GetDataPath())
	}
  if !is.IsScriptsPathValid() {
		return fmt.Errorf("chemin scripts invalide: %s", is.GetDataPath())
	}

	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case ".zip":
		return utils.ExtractZip(ctx, is.GetScriptsPath(), is.GetDataPath(), archivePath, callback)
	case ".rar":
		return utils.ExtractRar(ctx, is.GetScriptsPath(), is.GetDataPath(), archivePath, callback)
	case ".7z":
		return fmt.Errorf("format 7z non supporté dans cette version")
	default:
		return fmt.Errorf("format d'archive non supporté: %s", ext)
	}
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
