// models/mod.go
package models

import "time"

// Mod représente un mod disponible à l'installation
type Mod struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	DownloadURL string    `json:"download_url"`
	FileSize    int64     `json:"file_size"`
	Checksum    string    `json:"checksum"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// Métadonnées d'installation
	InstallPath  string   `json:"install_path"`
	Dependencies []string `json:"dependencies"`
	Conflicts    []string `json:"conflicts"`
}

// IsInstalled vérifie si le mod est déjà installé
func (m *Mod) IsInstalled(gamePath string) bool {
	// Logique pour vérifier l'installation
	return false // TODO: implémenter
}

// GetInstallSize retourne la taille d'installation estimée
func (m *Mod) GetInstallSize() int64 {
	return m.FileSize
}

// ModList représente une liste de mods avec métadonnées
type ModList struct {
	Mods      []Mod     `json:"mods"`
	Total     int       `json:"total"`
	Page      int       `json:"page"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Installation représente l'état d'une installation
type Installation struct {
	ModID      string    `json:"mod_id"`
	Status     Status    `json:"status"`
	Progress   float64   `json:"progress"`
	Error      string    `json:"error,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// Status représente les différents états d'installation
type Status int

const (
	StatusPending Status = iota
	StatusDownloading
	StatusExtracting
	StatusInstalling
	StatusCompleted
	StatusFailed
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "En attente"
	case StatusDownloading:
		return "Téléchargement"
	case StatusExtracting:
		return "Extraction"
	case StatusInstalling:
		return "Installation"
	case StatusCompleted:
		return "Terminé"
	case StatusFailed:
		return "Échec"
	default:
		return "Inconnu"
	}
}
