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

type MainWindow struct {
	app    fyne.App
	window fyne.Window
	config *config.Config
	
	downloader *services.DownloadService
	installer  *services.InstallerService
	
	gamePathEntry    *widget.Entry
	scriptsPathEntry *widget.Entry
	modList          *widget.List
	progressBar      *widget.ProgressBar
	statusLabel      *widget.Label
	installBtn       *widget.Button
	backupCheck      *widget.Check
	
	availableMods map[string]models.Mod
	modKeys       []string
	selectedMods  map[string]bool
}

func NewMainWindow(app fyne.App, cfg *config.Config) *MainWindow {
	window := app.NewWindow("Mod Installer")
	window.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))
	
	fmt.Println("Chargement des mods...")
	availableMods, err := api.FetchAllModMeta()
	if err != nil {
		fmt.Printf("Erreur API: %v\n", err)
		availableMods = getExampleModsMap()
	}
	
	modKeys := make([]string, 0, len(availableMods))
	for key := range availableMods {
		modKeys = append(modKeys, key)
	}
	
	mw := &MainWindow{
		app:           app,
		window:        window,
		config:        cfg,
		downloader:    services.NewDownloadService(cfg.TempPath, cfg.VerifyChecksums),
		installer:     services.NewInstallerService(cfg),
		availableMods: availableMods,
		modKeys:       modKeys,
		selectedMods:  make(map[string]bool),
	}
	
	mw.setupUI()
	return mw
}

func (mw *MainWindow) setupUI() {
	title := widget.NewLabel("Installeur de Mods")
	title.TextStyle.Bold = true
	
	// Chemin du jeu
	mw.gamePathEntry = widget.NewEntry()
	mw.gamePathEntry.SetText(mw.config.GamePath)
	mw.gamePathEntry.OnChanged = func(text string) {
		mw.config.SetGamePath(text)
		mw.installer = services.NewInstallerService(mw.config)
		mw.updateGamePathValidation()
	}
	
	browseGameBtn := widget.NewButton("Parcourir...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				path := uri.Path()
				mw.gamePathEntry.SetText(path)
				mw.config.SetGamePath(path)
				mw.installer = services.NewInstallerService(mw.config)
				mw.updateGamePathValidation()
			}
		}, mw.window)
	})
	
	// Chemin des scripts
	mw.scriptsPathEntry = widget.NewEntry()
	mw.scriptsPathEntry.SetText(mw.config.ScriptsPath)
	mw.scriptsPathEntry.OnChanged = func(text string) {
		mw.config.SetScriptsPath(text)
		mw.installer = services.NewInstallerService(mw.config)
	}
	
	browseScriptsBtn := widget.NewButton("Parcourir...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				path := uri.Path()
				mw.scriptsPathEntry.SetText(path)
				mw.config.SetScriptsPath(path)
				mw.installer = services.NewInstallerService(mw.config)
			}
		}, mw.window)
	})
	
	mw.backupCheck = widget.NewCheck("Créer des backups", func(checked bool) {
		mw.installer = services.NewInstallerService(mw.config)
	})
	mw.backupCheck.SetChecked(false)
	
	mw.modList = widget.NewList(
		func() int { return len(mw.modKeys) },
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			nameLabel := widget.NewLabel("Nom")
			descLabel := widget.NewLabel("Description")
			sizeLabel := widget.NewLabel("Taille")
			statusLabel := widget.NewLabel("")
			
			return container.NewVBox(
				container.NewHBox(check, nameLabel, widget.NewSeparator(), sizeLabel),
				descLabel, statusLabel,
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(mw.modKeys) { return }
			
			modKey := mw.modKeys[id]
			mod, exists := mw.availableMods[modKey]
			if !exists { return }
			
			vbox := item.(*fyne.Container)
			topRow := vbox.Objects[0].(*fyne.Container)
			check := topRow.Objects[0].(*widget.Check)
			nameLabel := topRow.Objects[1].(*widget.Label)
			sizeLabel := topRow.Objects[3].(*widget.Label)
			descLabel := vbox.Objects[1].(*widget.Label)
			statusLabel := vbox.Objects[2].(*widget.Label)
			
			nameLabel.SetText(fmt.Sprintf("%s v%s", mod.Name, mod.Version))
			nameLabel.TextStyle.Bold = true
			descLabel.SetText(mod.Description)
			sizeLabel.SetText(formatFileSize(mod.FileSize))
			
			statusText := ""
			if mw.downloader.IsModCached(&mod) {
				statusText += "📦 Cache"
			}
			if installed, _ := mw.installer.GetInstallationStatus(&mod); installed {
				if statusText != "" { statusText += " | " }
				statusText += "✅ Installé"
			}
			statusLabel.SetText(statusText)
			
			check.OnChanged = nil
			check.SetChecked(mw.selectedMods[modKey])
			check.OnChanged = func(checked bool) {
				mw.toggleModSelection(modKey, checked)
			}
		},
	)
	
	mw.progressBar = widget.NewProgressBar()
	mw.progressBar.Hide()
	mw.statusLabel = widget.NewLabel("Prêt")
	
	mw.installBtn = widget.NewButton("Installer sélectionnés", mw.installSelectedMods)
	refreshBtn := widget.NewButton("Actualiser", mw.refreshModList)
	backupsBtn := widget.NewButton("Backups", mw.showBackupManager)
	cacheBtn := widget.NewButton("Cache", mw.showCacheManager)
	
	topSection := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel("Chemin du jeu:"),
		container.NewBorder(nil, nil, nil, browseGameBtn, mw.gamePathEntry),
		widget.NewLabel("Chemin des scripts:"),
		container.NewBorder(nil, nil, nil, browseScriptsBtn, mw.scriptsPathEntry),
		mw.backupCheck,
	)
	
	bottomSection := container.NewVBox(
		mw.progressBar,
		mw.statusLabel,
		container.NewHBox(mw.installBtn, refreshBtn, backupsBtn, cacheBtn),
	)
	
	modListContainer := container.NewBorder(
		widget.NewLabel("Mods disponibles:"), nil, nil, nil, mw.modList,
	)
	
	lowerSection := container.NewVSplit(modListContainer, bottomSection)
	lowerSection.SetOffset(0.75)
	
	mainContent := container.NewVSplit(topSection, lowerSection)
	mainContent.SetOffset(0.3) // Augmenté légèrement pour faire de la place aux nouveaux champs
	
	mw.window.SetContent(mainContent)
}

func (mw *MainWindow) ShowAndRun() {
	mw.window.ShowAndRun()
}

func (mw *MainWindow) toggleModSelection(modKey string, selected bool) {
	mw.selectedMods[modKey] = selected
	count := 0
	for _, sel := range mw.selectedMods {
		if sel { count++ }
	}
	if count == 0 {
		mw.statusLabel.SetText("Prêt")
	} else {
		mw.statusLabel.SetText(fmt.Sprintf("%d mod(s) sélectionné(s)", count))
	}
}

func (mw *MainWindow) updateGamePathValidation() {
	if mw.installer.IsGamePathValid() {
		mw.statusLabel.SetText("Chemin valide")
	} else if mw.gamePathEntry.Text != "" {
		mw.statusLabel.SetText("⚠️ Chemin invalide")
	}
}

func (mw *MainWindow) installSelectedMods() {
	selectedModKeys := make([]string, 0)
	for key, selected := range mw.selectedMods {
		if selected {
			selectedModKeys = append(selectedModKeys, key)
		}
	}
	
	if len(selectedModKeys) == 0 {
		dialog.ShowInformation("Aucune sélection", "Sélectionnez au moins un mod", mw.window)
		return
	}
	
	if !mw.installer.IsGamePathValid() {
		dialog.ShowError(fmt.Errorf("chemin invalide"), mw.window)
		return
	}
	
	fyne.Do(func() {
		mw.statusLabel.SetText("Préparation...")
		mw.progressBar.Show()
		mw.progressBar.SetValue(0)
		mw.installBtn.Disable()
	})
	
	go mw.performInstallation(selectedModKeys)
}

func (mw *MainWindow) performInstallation(modKeys []string) {
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
		if !exists { continue }
		
		fyne.Do(func() {
			mw.statusLabel.SetText(fmt.Sprintf("Téléchargement %s (%d/%d)", mod.Name, i+1, totalMods))
		})
		
		downloadProgressCallback := func(downloaded, total int64) {
			if total > 0 {
				modProgress := float64(downloaded) / float64(total)
				overallProgress := (float64(i)*2 + modProgress) / float64(totalMods*2)
				fyne.Do(func() { mw.progressBar.SetValue(overallProgress) })
			}
		}
		
		filePath, err := mw.downloader.DownloadMod(ctx, &mod, downloadProgressCallback)
		if err != nil {
			fyne.Do(func() {
				mw.statusLabel.SetText(fmt.Sprintf("Erreur téléchargement %s", mod.Name))
				dialog.ShowError(err, mw.window)
			})
			return
		}
		
		fyne.Do(func() {
			mw.statusLabel.SetText(fmt.Sprintf("Installation %s (%d/%d)", mod.Name, i+1, totalMods))
		})
		
		installProgressCallback := func(currentFile string, processed, total int) {
			installProgress := float64(processed) / float64(total)
			overallProgress := (float64(i)*2 + 1 + installProgress) / float64(totalMods*2)
			fyne.Do(func() {
				mw.progressBar.SetValue(overallProgress)
				if currentFile != "" {
					mw.statusLabel.SetText(fmt.Sprintf("Installation %s: %s", mod.Name, currentFile))
				}
			})
		}
		
		err = mw.installer.InstallMod(ctx, &mod, filePath, installProgressCallback)
		if err != nil {
			fyne.Do(func() {
				mw.statusLabel.SetText(fmt.Sprintf("Erreur installation %s", mod.Name))
				dialog.ShowError(err, mw.window)
			})
			return
		}
		
		overallProgress := float64(i+1) / float64(totalMods)
		fyne.Do(func() { mw.progressBar.SetValue(overallProgress) })
	}
	
	fyne.Do(func() {
		mw.statusLabel.SetText(fmt.Sprintf("Terminé (%d mods)", totalMods))
		mw.refreshModList()
		
		_, cacheSize, cacheCount, _ := mw.downloader.GetCacheInfo()
		backups, _ := mw.installer.GetBackups()
		
		message := fmt.Sprintf("Installation terminée!\nMods: %d\nCache: %s (%d fichiers)\nBackups: %d",
			totalMods, formatFileSize(cacheSize), cacheCount, len(backups))
		
		dialog.ShowInformation("Terminé", message, mw.window)
	})
}

func (mw *MainWindow) refreshModList() {
	availableMods, err := api.FetchAllModMeta()
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}
	
	mw.availableMods = availableMods
	mw.modKeys = make([]string, 0, len(availableMods))
	for key := range availableMods {
		mw.modKeys = append(mw.modKeys, key)
	}
	
	mw.selectedMods = make(map[string]bool)
	mw.modList.Refresh()
	mw.statusLabel.SetText("Prêt")
}

func (mw *MainWindow) showBackupManager() {
	backups, err := mw.installer.GetBackups()
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}
	
	if len(backups) == 0 {
		dialog.ShowInformation("Backups", "Aucun backup", mw.window)
		return
	}
	
	backupList := widget.NewList(
		func() int { return len(backups) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Backup"),
				widget.NewButton("Restaurer", nil),
				widget.NewButton("Supprimer", nil),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(backups) { return }
			
			backup := backups[id]
			cont := item.(*fyne.Container)
			label := cont.Objects[0].(*widget.Label)
			restoreBtn := cont.Objects[1].(*widget.Button)
			deleteBtn := cont.Objects[2].(*widget.Button)
			
			label.SetText(backup)
			
			restoreBtn.OnTapped = func() {
				dialog.ShowConfirm("Confirmer", 
					fmt.Sprintf("Restaurer '%s'?", backup),
					func(confirmed bool) {
						if confirmed {
							if err := mw.installer.RestoreBackup(backup); err != nil {
								dialog.ShowError(err, mw.window)
							} else {
								dialog.ShowInformation("Succès", "Restauré", mw.window)
							}
						}
					}, mw.window)
			}
			
			deleteBtn.OnTapped = func() {
				dialog.ShowConfirm("Confirmer", 
					fmt.Sprintf("Supprimer '%s'?", backup),
					func(confirmed bool) {
						if confirmed {
							if err := mw.installer.DeleteBackup(backup); err != nil {
								dialog.ShowError(err, mw.window)
							} else {
								dialog.ShowInformation("Succès", "Supprimé", mw.window)
								mw.showBackupManager()
							}
						}
					}, mw.window)
			}
		},
	)
	
	backupWindow := mw.app.NewWindow("Backups")
	backupWindow.Resize(fyne.NewSize(600, 400))
	backupWindow.SetContent(container.NewBorder(
		widget.NewLabel("Backups:"), nil, nil, nil, backupList,
	))
	backupWindow.Show()
}

func (mw *MainWindow) showCacheManager() {
	cacheDir, cacheSize, cacheCount, err := mw.downloader.GetCacheInfo()
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}
	
	message := fmt.Sprintf("Dossier: %s\nTaille: %s\nFichiers: %d",
		cacheDir, formatFileSize(cacheSize), cacheCount)
	
	dialog.ShowConfirm("Cache", 
		message+"\n\nVider le cache?",
		func(confirmed bool) {
			if confirmed {
				if err := mw.downloader.ClearCache(); err != nil {
					dialog.ShowError(err, mw.window)
				} else {
					dialog.ShowInformation("Succès", "Cache vidé", mw.window)
				}
			}
		}, mw.window)
}

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
			ID: "mod1", Name: "UI Mod", Version: "1.2.3",
			Description: "Améliore l'interface", FileSize: 2048576,
			DownloadURL: "https://example.com/mod1.zip",
		},
		"mod2": {
			ID: "mod2", Name: "Contenu Extra", Version: "2.0.1", 
			Description: "Mod de contenu", FileSize: 10485760,
			DownloadURL: "https://example.com/mod2.zip",
		},
	}
}
