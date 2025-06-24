// services/vanilla.go
package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mod-installer/models"
	"mod-installer/utils"
)

// VanillaService gère les fichiers vanilla du jeu
type VanillaService struct {
	ScriptsPath, GamePath, CacheDir  string
  vanillaFiles, vanillaScripts []string
}

func NewVanillaService(gamePath, scriptsPath, cacheDir string) *VanillaService {
	return &VanillaService{
		GamePath: gamePath,
    ScriptsPath : scriptsPath,
		CacheDir: filepath.Join(cacheDir, "vanilla"),
    vanillaFiles: []string{
		  "data/media.pack",
      "data/boot.pack",
	  },
    vanillaScripts: []string{
		  "scripts/user.script.txt",
	  },
	}
}

// GetVanillaMod retourne les fichiers vanilla comme un seul mod
func (vs *VanillaService) GetVanillaMod() (models.Mod, error) {
  
  var mod models.Mod
	var totalSize int64
	existingFiles := make([]string, 0)

  if strings.HasSuffix(vs.GamePath, "Napoleon Total War") {

    fmt.Println("Getting vanilla files...")

  
	for _, file := range vs.vanillaFiles {
		fullPath := filepath.Join(vs.GamePath, file)
		if !utils.FileExists(fullPath) {
      fmt.Println("NOT found : ", file)
      continue
		}
		
		size, err := utils.GetFileSize(fullPath)
		if err != nil {
      fmt.Println("NOT found : ", file)
      continue
		}
		
		totalSize += size
		existingFiles = append(existingFiles, file)
    fmt.Println("Found : ", file)
	}
  }
	
	if len(existingFiles) > 0 {
		mod = models.Mod{
			ID:          "vanilla_pack",
			Name:        "Vanilla Files",
			Version:     "original",
			Description: fmt.Sprintf("Original game files (%d files)", len(existingFiles)),
			FileSize:    totalSize,
			DownloadURL: "", // Pas d'URL pour les fichiers vanilla
			Checksum:    "vanilla",
		}	
	} else {
    mod = models.Mod{
			ID:          "vanilla_pack",
			Name:        "No vanilla files found",
			Version:     "original",
			Description: fmt.Sprintf("Original game files (%d files)", len(existingFiles)),
			FileSize:    0,
			DownloadURL: "", // Pas d'URL pour les fichiers vanilla
			Checksum:    "vanilla",
		}
  }

  if !vs.IsVanillaBacked(&mod) {
    vs.AutoBackupVanillaFiles()
  }
	
	return mod, nil
}

// BackupVanillaFile sauvegarde un fichier vanilla dans le cache
func (vs *VanillaService) BackupVanillaFile(filePath string) error {
  if !utils.FileExists(filePath) {
		return fmt.Errorf("fichier vanilla introuvable: %s", filePath)
	}
	
	// Créer le répertoire de cache si nécessaire
	if err := utils.EnsureDirectoryExists(vs.CacheDir); err != nil {
		return err
	}
	var relPath string
  var err error

	// Nom du fichier de sauvegarde basé sur le chemin relatif
	relPath, err = utils.GetRelativePath(vs.GamePath, filePath)

	if err != nil {
		return err
	}
	
	backupPath := filepath.Join(vs.CacheDir, strings.ReplaceAll(relPath, string(os.PathSeparator), "_"))
	
	return utils.CopyFile(filePath, backupPath)
}

// RestoreVanillaFile restaure tous les fichiers vanilla depuis le cache
func (vs *VanillaService) RestoreVanillaFile(mod *models.Mod) error {
  fmt.Println("Trying to restore vanilla files...")
	if mod.ID != "vanilla_pack" {
		return fmt.Errorf("mod vanilla invalide: %s", mod.ID)
	}
	
	restoredCount := 0
	for _, file := range vs.vanillaFiles {
		// Nom du fichier de backup
		fileName := filepath.Base(file)
		backupPath := filepath.Join(vs.CacheDir, strings.ReplaceAll(file, string(os.PathSeparator), "_"))
		

		if !utils.FileExists(backupPath) {
      
			continue // Skip si pas de backup
		}
		
		// Chemin de destination
		destPath := filepath.Join(vs.GamePath, file)
		
		if err := utils.CopyFile(backupPath, destPath); err != nil {
			return fmt.Errorf("erreur restauration %s: %v", fileName, err)
		}
		
		restoredCount++
	}

  for _, file := range vs.vanillaScripts {
    delPath := filepath.Join(vs.GamePath, file)
    err := os.Remove(delPath)
    if err != nil {
        fmt.Println("Erreur lors de la suppression :", err)
    } else {
        fmt.Println("Script supprimé avec succès")
    }
  }

	
	if restoredCount == 0 {
		return fmt.Errorf("aucun fichier vanilla à restaurer")
	}
	
	return nil
}

// IsVanillaBacked vérifie si les fichiers vanilla sont sauvegardés
func (vs *VanillaService) IsVanillaBacked(mod *models.Mod) bool {
  //fmt.Println("Checking if mod is vanilla files...")
	if mod.ID != "vanilla_pack" {
		return false
	}
	
  backedCount := 0
	for _, file := range vs.vanillaFiles {
		backupPath := filepath.Join(vs.CacheDir, strings.ReplaceAll(file, string(os.PathSeparator), "_"))
		if utils.FileExists(backupPath) {
			backedCount++
		}
	}
	
	return backedCount > 0 // Au moins un fichier sauvegardé
}

// AutoBackupVanillaFiles sauvegarde automatiquement les fichiers vanilla importants
func (vs *VanillaService) AutoBackupVanillaFiles() error {
  fmt.Println("=======================================")
  fmt.Println("Starting vanilla files backup...")
  if !strings.HasSuffix(vs.GamePath, "Napoleon Total War") {
    fmt.Println("Wrong game path")
  	fmt.Println("=======================================")
    return nil
  }
  backedCount := 0
	
	if err := utils.EnsureDirectoryExists(vs.CacheDir); err != nil {
		return err
	}
	
	for _, file := range vs.vanillaFiles {
		fullPath := filepath.Join(vs.GamePath, file)
    fmt.Println("Checking if", file, "needs backup...")
		if !utils.FileExists(fullPath) {
      fmt.Println("File not found", fullPath)
			continue
		}
		
		backupPath := filepath.Join(vs.CacheDir, strings.ReplaceAll(file, string(os.PathSeparator), "_"))
		
		// Ne pas re-sauvegarder si déjà fait
		if utils.FileExists(backupPath) {
      fmt.Println("Already backed up in", backupPath)
			continue
		} else {
      backedCount++
    }
		
		if err := utils.CopyFile(fullPath, backupPath); err != nil {
			return fmt.Errorf("erreur backup %s: %v", file, err)
		} else {
      fmt.Println("Backed to", backupPath);
    }
	}

  fmt.Println("Backed up", backedCount, "files")
	fmt.Println("=======================================")

	return nil
}
