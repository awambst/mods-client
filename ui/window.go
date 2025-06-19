// ui/window.go
package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"mod-installer/config"
	"mod-installer/models"
	"mod-installer/services"
	"mod-installer/api"
)

// MainWindow représente la fenêtre principale de l'application
type MainWindow struct {
	app    fyne.App
	window fyne.Window
	config *config.Config
	
	// Services
	downloader *services.DownloadService
	
	// UI Components
	gamePathEntry *widget.Entry
	modList       *widget.List
	progressBar   *widget.ProgressBar
	statusLabel   *widget.Label
	installBtn    *widget.Button
	
	// Data
	availableMods map[string]models.Mod
	modKeys       []string // Pour maintenir l'ordre des mods
	selectedMods  map[string]bool
}

// NewMainWindow crée une nouvelle fenêtre principale
func NewMainWindow(app fyne.App, cfg *config.Config) *MainWindow {
	window := app.NewWindow("Mod Installer")
	window.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))
	
	// Charger les mods depuis l'API
	fmt.Println("Chargement des mods depuis l'API...")
	availableMods, err := api.FetchAllModMeta()
	if err != nil {
		fmt.Printf("Erreur lors du chargement des mods: %v\n", err)
		fmt.Println("Utilisation des mods d'exemple à la place")
		// En cas d'erreur, utiliser des mods d'exemple
		availableMods = getExampleModsMap()
	}
	
	fmt.Printf("Nombre de mods chargés: %d\n", len(availableMods))
	for key, mod := range availableMods {
		fmt.Printf("- %s: %s v%s\n", key, mod.Name, mod.Version)
	}
	
	// Créer une slice des clés pour maintenir l'ordre
	modKeys := make([]string, 0, len(availableMods))
	for key := range availableMods {
		modKeys = append(modKeys, key)
	}
	
	mw := &MainWindow{
		app:           app,
		window:        window,
		config:        cfg,
		downloader:    services.NewDownloadService(cfg.TempPath, cfg.VerifyChecksums),
		availableMods: availableMods,
		modKeys:       modKeys,
		selectedMods:  make(map[string]bool),
	}
	
	mw.setupUI()
	return mw
}

// setupUI configure l'interface utilisateur
func (mw *MainWindow) setupUI() {
	// Titre
	title := widget.NewLabel("Installeur de Mods")
	title.TextStyle.Bold = true
	
	// Configuration du chemin du jeu
	gamePathSection := mw.createGamePathSection()
	
	// Section de progression
	progressSection := mw.createProgressSection()
	
	// Boutons d'action
	actionSection := mw.createActionSection()
	
	// Partie haute (compacte)
	topSection := container.NewVBox(
		title,
		widget.NewSeparator(),
		gamePathSection,
	)
	
	// Partie basse (compacte)
	bottomSection := container.NewVBox(
		progressSection,
		actionSection,
	)
	
	// Section des mods
	modListLabel := widget.NewLabel("Mods disponibles:")
	modListContainer := container.NewBorder(
		modListLabel, // top
		nil,          // bottom
		nil,          // left
		nil,          // right
		mw.createModList(), // center
	)
	
	// Split 1: partie haute vs (mods + bas)
	lowerSection := container.NewVSplit(
		modListContainer,
		bottomSection,
	)
	lowerSection.SetOffset(0.8) // 80% pour les mods, 20% pour les contrôles
	
	// Split 2: titre/config vs (mods + bas)
	mainContent := container.NewVSplit(
		topSection,
		lowerSection,
	)
	mainContent.SetOffset(0.2) // 20% pour le haut, 80% pour le reste
	
	mw.window.SetContent(mainContent)
}

// createGamePathSection crée la section de sélection du chemin du jeu
func (mw *MainWindow) createGamePathSection() *fyne.Container {
	mw.gamePathEntry = widget.NewEntry()
	mw.gamePathEntry.SetText(mw.config.GamePath)
	mw.gamePathEntry.SetPlaceHolder("Chemin vers le dossier du jeu")
	
	browseBtn := widget.NewButton("Parcourir...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				path := uri.Path()
				mw.gamePathEntry.SetText(path)
				mw.config.SetGamePath(path) // Sauvegarder automatiquement
			}
		}, mw.window)
	})
	
	return container.NewVBox(
		widget.NewLabel("Chemin du jeu:"),
		container.NewBorder(nil, nil, nil, browseBtn, mw.gamePathEntry),
	)
}

// createModList crée la liste des mods
func (mw *MainWindow) createModList() *widget.List {
	mw.modList = widget.NewList(
		func() int { return len(mw.modKeys) },
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			nameLabel := widget.NewLabel("Nom du mod")
			descLabel := widget.NewLabel("Description")
			sizeLabel := widget.NewLabel("Taille")
			
			return container.NewVBox(
				container.NewHBox(check, nameLabel, widget.NewSeparator(), sizeLabel),
				descLabel,
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(mw.modKeys) {
				return
			}
			
			modKey := mw.modKeys[id]
			mod, exists := mw.availableMods[modKey]
			if !exists {
				return
			}
			
			vbox := item.(*fyne.Container)
			topRow := vbox.Objects[0].(*fyne.Container)
			
			check := topRow.Objects[0].(*widget.Check)
			nameLabel := topRow.Objects[1].(*widget.Label)
			sizeLabel := topRow.Objects[3].(*widget.Label)
			descLabel := vbox.Objects[1].(*widget.Label)
			
			// Mise à jour des labels
			nameLabel.SetText(fmt.Sprintf("%s v%s", mod.Name, mod.Version))
			nameLabel.TextStyle.Bold = true
			descLabel.SetText(mod.Description)
			sizeLabel.SetText(formatFileSize(mod.FileSize))
			
			// Gestion de la sélection
			check.OnChanged = nil
			check.SetChecked(mw.selectedMods[modKey])
			check.OnChanged = func(checked bool) {
				mw.toggleModSelection(modKey, checked)
			}
		},
	)
	
	return mw.modList
}

// createProgressSection crée la section de progression
func (mw *MainWindow) createProgressSection() *fyne.Container {
	mw.progressBar = widget.NewProgressBar()
	mw.progressBar.Hide()
	
	mw.statusLabel = widget.NewLabel("Prêt")
	
	return container.NewVBox(
		mw.progressBar,
		mw.statusLabel,
	)
}

// createActionSection crée la section des boutons d'action
func (mw *MainWindow) createActionSection() *fyne.Container {
	mw.installBtn = widget.NewButton("Installer les mods sélectionnés", mw.installSelectedMods)
	
	refreshBtn := widget.NewButton("Actualiser", func() {
		mw.refreshModList()
	})
	
	return container.NewHBox(
		mw.installBtn,
		refreshBtn,
	)
}

// ShowAndRun affiche la fenêtre et lance l'application
func (mw *MainWindow) ShowAndRun() {
	mw.window.ShowAndRun()
}

// Méthodes utilitaires
func (mw *MainWindow) toggleModSelection(modKey string, selected bool) {
	mw.selectedMods[modKey] = selected
	mw.updateStatus()
}

func (mw *MainWindow) updateStatus() {
	count := 0
	for _, selected := range mw.selectedMods {
		if selected {
			count++
		}
	}
	
	if count == 0 {
		mw.statusLabel.SetText("Prêt")
	} else {
		mw.statusLabel.SetText(fmt.Sprintf("%d mod(s) sélectionné(s)", count))
	}
}

func (mw *MainWindow) installSelectedMods() {
	selectedCount := 0
	selectedModKeys := make([]string, 0)
	
	for key, selected := range mw.selectedMods {
		if selected {
			selectedCount++
			selectedModKeys = append(selectedModKeys, key)
		}
	}
	
	if selectedCount == 0 {
		dialog.ShowInformation("Aucune sélection", "Veuillez sélectionner au moins un mod", mw.window)
		return
	}
	
	// Vérifier le chemin du jeu
	gamePath := mw.gamePathEntry.Text
	if gamePath == "" {
		dialog.ShowError(fmt.Errorf("chemin du jeu non défini"), mw.window)
		return
	}
	
	// Démarrer l'installation asynchrone
	fyne.Do(func() {
		mw.statusLabel.SetText("Préparation de l'installation...")
		mw.progressBar.Show()
		mw.progressBar.SetValue(0)
		mw.installBtn.Disable()
	})
	
	go mw.performInstallation(selectedModKeys, gamePath)
}

// performInstallation effectue l'installation des mods sélectionnés
func (mw *MainWindow) performInstallation(modKeys []string, gamePath string) {
	defer func() {
		fyne.Do(func() {
			mw.progressBar.Hide()
			mw.installBtn.Enable()
		})
	}()
	
	totalMods := len(modKeys)
	ctx := context.Background()
	
	for i, modKey := range modKeys {
		mod, exists := mw.availableMods[modKey]
		if !exists {
			continue
		}
		
		// Mettre à jour le statut (thread-safe)
		fyne.Do(func() {
			mw.statusLabel.SetText(fmt.Sprintf("Installation de %s (%d/%d)...", mod.Name, i+1, totalMods))
		})
		
		// Callback de progression pour ce mod
		progressCallback := func(downloaded, total int64) {
			if total > 0 {
				modProgress := float64(downloaded) / float64(total)
				overallProgress := (float64(i) + modProgress) / float64(totalMods)
				
				// Mettre à jour la barre de progression (thread-safe)
				fyne.Do(func() {
					mw.progressBar.SetValue(overallProgress)
				})
			}
		}
		
		// Télécharger le mod
		filePath, err := mw.downloader.DownloadMod(ctx, &mod, progressCallback)
		if err != nil {
			// Afficher l'erreur (thread-safe)
			fyne.Do(func() {
				mw.statusLabel.SetText(fmt.Sprintf("Erreur lors du téléchargement de %s: %v", mod.Name, err))
				dialog.ShowError(fmt.Errorf("erreur installation %s: %w", mod.Name, err), mw.window)
			})
			return
		}
		
		// TODO: Ici vous pourriez ajouter l'extraction et l'installation
		fmt.Printf("Fichier téléchargé: %s\n", filePath)
		
		// Mettre à jour la progression (thread-safe)
		overallProgress := float64(i+1) / float64(totalMods)
		fyne.Do(func() {
			mw.progressBar.SetValue(overallProgress)
		})
	}
	
	// Installation terminée (thread-safe)
	fyne.Do(func() {
		mw.statusLabel.SetText(fmt.Sprintf("Installation terminée (%d mods)", totalMods))
		
		// Afficher les informations du cache
		if cacheDir, size, count, err := mw.downloader.GetCacheInfo(); err == nil {
			sizeStr := formatFileSize(size)
			message := fmt.Sprintf("Installation terminée!\n\nCache: %s\nTaille: %s (%d fichiers)", 
				cacheDir, sizeStr, count)
			dialog.ShowInformation("Installation terminée", message, mw.window)
		}
	})
}

func (mw *MainWindow) refreshModList() {
	// Recharger depuis l'API
	availableMods, err := api.FetchAllModMeta()
	if err != nil {
		// En cas d'erreur, garder les mods actuels
		dialog.ShowError(err, mw.window)
		return
	}
	
	mw.availableMods = availableMods
	
	// Recréer la liste des clés
	mw.modKeys = make([]string, 0, len(availableMods))
	for key := range availableMods {
		mw.modKeys = append(mw.modKeys, key)
	}
	
	// Réinitialiser les sélections
	mw.selectedMods = make(map[string]bool)
	mw.modList.Refresh()
	mw.updateStatus()
}

// Fonctions utilitaires
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func getExampleModsMap() map[string]models.Mod {
	return map[string]models.Mod{
		"mod1": {
			ID:          "mod1",
			Name:        "Example mod 1",
			Version:     "1.2.3",
			Description: "Améliore l'interface utilisateur du jeu",
			FileSize:    2048576, // 2MB
			DownloadURL: "https://example.com/mod1.zip",
		},
		"mod2": {
			ID:          "mod2",
			Name:        "Contenu Extra",
			Version:     "2.0.1",
			Description: "Example mod 2",
			FileSize:    10485760, // 10MB
			DownloadURL: "https://example.com/mod2.zip",
		},
	}
}
