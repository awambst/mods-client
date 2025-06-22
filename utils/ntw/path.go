package ntw

import (
	"path/filepath"
	"strings"
)


// GetDestinationPath détermine le dossier de destination en fonction de l'extension du fichier
func GetDestinationPath(scriptsPath, gamePath, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	
	switch ext {
	case ".txt":
		return scriptsPath
//	case ".pack":
//		return config.GetDataPath()
	default:
		// Pour les autres fichiers, on utilise le dossier data par défaut
		return gamePath
	}
}
