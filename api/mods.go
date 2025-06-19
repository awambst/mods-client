package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mod-installer/models"
)

type GitHubTreeResponse struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
}

func FetchAllModMeta() (map[string]models.Mod, error) {
	treeURL := "https://api.github.com/repos/awambst/mods-meta/git/trees/main?recursive=1"

	resp, err := http.Get(treeURL)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération de l'arbre GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur HTTP: %d", resp.StatusCode)
	}

	var tree GitHubTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage JSON: %w", err)
	}

	mods := make(map[string]models.Mod)

	fmt.Printf("Nombre de fichiers trouvés dans l'arbre: %d\n", len(tree.Tree))

	for _, item := range tree.Tree {
		fmt.Printf("Fichier trouvé: %s (type: %s)\n", item.Path, item.Type)
		
		if strings.HasSuffix(item.Path, ".json") && item.Type == "blob" {
			parts := strings.Split(item.Path, "/")
			fmt.Printf("Parties du chemin: %v\n", parts)
			
			if len(parts) < 2 {
				fmt.Printf("Chemin trop court, ignoré: %s\n", item.Path)
				continue // structure incorrecte, ignorer
			}
			
			// Construire une clé unique pour le mod basée sur le chemin complet
			// En excluant l'extension .json
			pathWithoutExt := strings.TrimSuffix(item.Path, ".json")
			modKey := strings.ReplaceAll(pathWithoutExt, "/", "_")

			url := fmt.Sprintf("https://raw.githubusercontent.com/awambst/mods-meta/main/%s", item.Path)
			fmt.Printf("Tentative de chargement du mod: %s depuis %s\n", modKey, url)
			
			meta, err := fetchOneModMeta(url)
			if err != nil {
				// Log l'erreur mais continue avec les autres mods
				fmt.Printf("Erreur lors du chargement du mod %s: %v\n", modKey, err)
				continue
			}

			// Enrichir les métadonnées basées sur le chemin
			if meta.ID == "" {
				meta.ID = modKey
			}
			if meta.Name == "" {
				// Utiliser le nom du dossier parent comme nom
				if len(parts) >= 2 {
					meta.Name = strings.ToUpper(parts[1]) // "fcn" -> "FCN"
				} else {
					meta.Name = "Mod sans nom"
				}
			}
			if meta.Version == "" {
				// Utiliser le nom du fichier (sans .json) comme version
				filename := parts[len(parts)-1]
				meta.Version = strings.TrimSuffix(filename, ".json") // "8.2.0.json" -> "8.2.0"
			}
			if meta.Description == "" {
				meta.Description = fmt.Sprintf("Mod %s pour %s", meta.Name, strings.ToUpper(parts[0]))
			}

			mods[modKey] = meta
			fmt.Printf("Mod chargé avec succès: %s (%s)\n", modKey, meta.Name)
		}
	}

	fmt.Printf("Nombre total de mods chargés: %d\n", len(mods))

	if len(mods) == 0 {
		return nil, fmt.Errorf("aucun mod trouvé")
	}

	return mods, nil
}

// Structure pour le format JSON de votre repository
type ModMetaFormat struct {
	Metadata struct {
		Link string `json:"link"`
		Size string `json:"size"`
		Day  string `json:"day"`
	} `json:"metadata"`
	Installation []string `json:"installation"`
}

func fetchOneModMeta(url string) (models.Mod, error) {
	fmt.Printf("Téléchargement de: %s\n", url)
	
	resp, err := http.Get(url)
	if err != nil {
		return models.Mod{}, fmt.Errorf("erreur lors de la requête HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Mod{}, fmt.Errorf("erreur HTTP %d pour l'URL: %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Mod{}, fmt.Errorf("erreur lors de la lecture du corps de la réponse: %w", err)
	}

	fmt.Printf("Contenu JSON reçu: %s\n", string(body))

	// D'abord essayer de décoder dans le format de votre repository
	var metaFormat ModMetaFormat
	if err := json.Unmarshal(body, &metaFormat); err != nil {
		return models.Mod{}, fmt.Errorf("erreur lors du décodage JSON du mod: %w", err)
	}

	// Convertir vers le format models.Mod
	mod := models.Mod{
		DownloadURL: metaFormat.Metadata.Link,
		CreatedAt:   parseDate(metaFormat.Metadata.Day),
	}

	// Convertir la taille (string vers int64)
	if sizeStr := metaFormat.Metadata.Size; sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			// Supposer que la taille est en MB, convertir en bytes
			mod.FileSize = size * 1024 * 1024
		}
	}

	return mod, nil
}

// Fonction utilitaire pour parser la date
func parseDate(dateStr string) time.Time {
	// Format: "01/02/2003" (MM/DD/YYYY)
	if t, err := time.Parse("01/02/2006", dateStr); err == nil {
		return t
	}
	return time.Now() // Fallback sur la date actuelle
}
