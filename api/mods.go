package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
		return nil, err
	}
	defer resp.Body.Close()

	var tree GitHubTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return nil, err
	}

	//mods := make([]models.Mod, 55)
  mods := make(map[string]models.Mod)

	for _, item := range tree.Tree {
		if strings.HasSuffix(item.Path, ".json") {
			parts := strings.Split(item.Path, "/")
			if len(parts) != 3 {
				continue // mauvaise structure
			}
			//jeu := parts[0]
			mod := parts[1]

			url := fmt.Sprintf("https://raw.githubusercontent.com/awambst/mods-meta/main/%s", item.Path)
			meta, err := fetchOneModMeta(url)
			if err != nil {
				continue
			}

      //if _, ok := mods[jeu]; !ok {
			//	mods.append(jeu) 
			//}
			mods[mod] = meta
		}
	}

	return mods, nil
}

func fetchOneModMeta(url string) (models.Mod, error) {
	resp, err := http.Get(url)
	if err != nil {
		return models.Mod{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.Mod{}, err
	}

	var meta models.Mod
	if err := json.Unmarshal(body, &meta); err != nil {
		return models.Mod{}, err
	}
	return meta, nil
}

