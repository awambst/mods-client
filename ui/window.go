// ui/window.go
package ui

import (
	//"context"
	"fmt"

	"fyne.io/fyne/v2"
	//"fyne.io/fyne/v2/app"
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
	selectedMods  []int
}

// NewMainWindow crée une nouvelle fenêtre principale
func NewMainWindow(app fyne.App, cfg *config.Config) *MainWindow {
	window := app.NewWindow("Mod Installer")
	window.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))
	
	mw := &MainWindow{
		app:           app,
		window:        window,
		config:        cfg,
		downloader:    services.NewDownloadService(cfg.TempPath, cfg.VerifyChecksums),
		availableMods: api.FetchAllModMeta(), // TODO: charger depuis l'API
		selectedMods:  make([]int, 0),
	}
	
	mw.setupUI()
	return mw
}

// ALTERNATIVE: setupUI avec VSplit vraiment redimensionnable
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
	
	// Section des mods (juste la liste, pas de VBox)
	modListLabel := widget.NewLabel("Mods disponibles:")
	modListContainer := container.NewBorder(
		modListLabel, // top
		nil,          // bottom
		nil,          // left
		nil,          // right
		mw.createModList(), // center - juste la liste
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
// createModList crée juste la liste sans container supplémentaire
func (mw *MainWindow) createModList() *widget.List {
	mw.modList = widget.NewList(
		func() int { return len(mw.availableMods) },
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
			mod := mw.availableMods[id]
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
			check.SetChecked(mw.isModSelected(id))
			check.OnChanged = func(checked bool) {
				mw.toggleModSelection(id, checked)
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
func (mw *MainWindow) isModSelected(id int) bool {
	for _, selectedID := range mw.selectedMods {
		if selectedID == id {
			return true
		}
	}
	return false
}

func (mw *MainWindow) toggleModSelection(id int, selected bool) {
	if selected {
		if !mw.isModSelected(id) {
			mw.selectedMods = append(mw.selectedMods, id)
		}
	} else {
		for i, selectedID := range mw.selectedMods {
			if selectedID == id {
				mw.selectedMods = append(mw.selectedMods[:i], mw.selectedMods[i+1:]...)
				break
			}
		}
	}
	mw.updateStatus()
}

func (mw *MainWindow) updateStatus() {
	count := len(mw.selectedMods)
	if count == 0 {
		mw.statusLabel.SetText("Prêt")
	} else {
		mw.statusLabel.SetText(fmt.Sprintf("%d mod(s) sélectionné(s)", count))
	}
}

func (mw *MainWindow) installSelectedMods() {
	if len(mw.selectedMods) == 0 {
		dialog.ShowInformation("Aucune sélection", "Veuillez sélectionner au moins un mod", mw.window)
		return
	}
	
	// TODO: Implémenter l'installation
	mw.statusLabel.SetText("Installation en cours...")
	mw.progressBar.Show()
	mw.installBtn.Disable()
	
	// Simulation pour l'exemple
	go func() {
		// Installation asynchrone ici
		// ...
		
		mw.progressBar.Hide()
		mw.installBtn.Enable()
		mw.statusLabel.SetText("Installation terminée")
	}()
}

func (mw *MainWindow) refreshModList() {
	// TODO: Recharger depuis l'API
	mw.selectedMods = make([]int, 0)
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

func getExampleMods() []models.Mod {
	return []models.Mod{
		{
			ID:          "mod1",
			Name:        "Example mod 1",
			Version:     "1.2.3",
			Description: "Améliore l'interface utilisateur du jeu",
			FileSize:    2048576, // 2MB
			DownloadURL: "https://example.com/mod1.zip",
		},
		{
			ID:          "mod2",
			Name:        "Contenu Extra",
			Version:     "2.0.1",
			Description: "Example mod 2",
			FileSize:    10485760, // 10MB
			DownloadURL: "https://example.com/mod2.zip",
		},
	}
}
