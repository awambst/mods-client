// config/config.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Config contient la configuration de l'application
type Config struct {
	// Chemins
	GamePath     string `json:"game_path"`
	ScriptsPath  string `json:"scripts_path"`  // Nouveau: chemin pour les scripts
	ModsPath     string `json:"mods_path"`
	TempPath     string `json:"temp_path"`
	ConfigPath   string `json:"config_path"`
	
	// API
	ModRepositoryURL string `json:"mod_repository_url"`
	APITimeout       int    `json:"api_timeout"`
	
	// Interface
	WindowWidth  int  `json:"window_width"`
	WindowHeight int  `json:"window_height"`
	DarkTheme    bool `json:"dark_theme"`
	
	// Installation
	MaxConcurrentDownloads int  `json:"max_concurrent_downloads"`
	VerifyChecksums        bool `json:"verify_checksums"`
	CreateBackups          bool `json:"create_backups"`
}

// Default retourne une configuration par défaut
func Default() *Config {
	homeDir, _ := os.UserHomeDir()

  base := ""

  switch runtime.GOOS {
	  case "windows":
		  if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
			  base = filepath.Join(appData, "ModInstaller", "cache")
		  }
	  case "linux":
		  if home := os.Getenv("HOME"); home != "" {
			  base = filepath.Join(home, ".cache", "mod-installer")
		  }
	  case "darwin":
		  if home := os.Getenv("HOME"); home != "" {
			  base =filepath.Join(home, "Library", "Caches", "ModInstaller")
		  }
  }
  
  if base == "" {
  	base = filepath.Join(os.TempDir(), "mod-installer-cache")
  }

	
	return &Config{
		GamePath:               filepath.Join(homeDir, "."),
		ScriptsPath:            filepath.Join(homeDir, "."),
    ModsPath:               filepath.Join(base, "mods"),
		TempPath:               filepath.Join(base, "temp"),
    ConfigPath:             filepath.Join(base, "config.json"),
		ModRepositoryURL:       "https://api.example.com/mods",
		APITimeout:             30,
		WindowWidth:            800,
		WindowHeight:           600,
		DarkTheme:              false,
		MaxConcurrentDownloads: 3,
		VerifyChecksums:        true,
		CreateBackups:          true,
	}
}

// Load charge la configuration depuis le fichier ou crée une config par défaut
func Load() (*Config, error) {
	cfg := Default()
	
	// Créer le dossier de config s'il n'existe pas
	configDir := filepath.Dir(cfg.ConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}
	
	// Charger depuis le fichier s'il existe
	if _, err := os.Stat(cfg.ConfigPath); err == nil {
		data, err := os.ReadFile(cfg.ConfigPath)
		if err != nil {
			return nil, err
		}
		
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}
	
	// Créer les dossiers nécessaires
	dirs := []string{cfg.ModsPath, cfg.TempPath, cfg.ScriptsPath}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	
	return cfg, nil
}

// Save sauvegarde la configuration
func (c *Config) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(c.ConfigPath, data, 0644)
}

// SetGamePath met à jour le chemin du jeu et sauvegarde
func (c *Config) SetGamePath(path string) error {
	c.GamePath = path
	// Mettre à jour aussi le chemin des scripts par défaut
	c.ScriptsPath = filepath.Join(path, "scripts")
	return c.Save()
}

// SetScriptsPath met à jour le chemin des scripts et sauvegarde
func (c *Config) SetScriptsPath(path string) error {
	c.ScriptsPath = path
	return c.Save()
}
