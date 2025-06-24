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
	
	downloader     *services.DownloadService
	installer      *services.InstallerService
	vanillaService *services.VanillaService  // Service séparé pour vanilla
	
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
	
	fmt.Println("Loading mods...")
	availableMods, err := api.FetchAllModMeta()
	if err != nil {
		fmt.Printf("API Error: %v\n", err)
		availableMods = getExampleModsMap()
	}
	
	mw := &MainWindow{
		app:            app,
		window:         window,
		config:         cfg,
		downloader:     services.NewDownloadService(cfg.TempPath, cfg.VerifyChecksums),
		installer:      services.NewInstallerService(cfg),
		vanillaService: services.NewVanillaService(cfg.GamePath, cfg.ScriptsPath, cfg.TempPath),
		availableMods:  availableMods,
		selectedMods:   make(map[string]bool),
	}
	
	mw.loadAllMods()
	mw.setupUI()
	return mw
}

func (mw *MainWindow) loadAllMods() {
	// Charger les mods vanilla
	if mw.vanillaService != nil {
		vanillaMod, err := mw.vanillaService.GetVanillaMod()
		if err == nil {
			// Fusionner avec les mods existants
			mw.availableMods[vanillaMod.Name] = vanillaMod
		}
	}
	
	// Créer la liste des clés
	mw.modKeys = make([]string, 0, len(mw.availableMods))
	for key := range mw.availableMods {
		mw.modKeys = append(mw.modKeys, key)
	}
}

func (mw *MainWindow) setupUI() {
	title := widget.NewLabel("Mod Installer")
	title.TextStyle.Bold = true
	
	// Game path
	mw.gamePathEntry = widget.NewEntry()
	mw.gamePathEntry.SetText(mw.config.GamePath)
	mw.gamePathEntry.OnChanged = func(text string) {
		mw.config.SetGamePath(text)
		mw.installer = services.NewInstallerService(mw.config)
		mw.vanillaService = services.NewVanillaService(text, mw.config.ScriptsPath, mw.config.TempPath)
		mw.loadAllMods()
		mw.updateGamePathValidation()
	}
	
	browseGameBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				path := uri.Path()
				mw.gamePathEntry.SetText(path)
				mw.config.SetGamePath(path)
				mw.installer = services.NewInstallerService(mw.config)
				mw.vanillaService = services.NewVanillaService(path, mw.config.ScriptsPath, mw.config.TempPath)
				mw.loadAllMods()
				mw.updateGamePathValidation()
			}
		}, mw.window)
	})
	
	// Scripts path
	mw.scriptsPathEntry = widget.NewEntry()
	mw.scriptsPathEntry.SetText(mw.config.ScriptsPath)
	mw.scriptsPathEntry.OnChanged = func(text string) {
		mw.config.SetScriptsPath(text)
		mw.vanillaService = services.NewVanillaService(mw.config.GamePath, text, mw.config.TempPath)
		mw.loadAllMods()
		mw.installer = services.NewInstallerService(mw.config)
	}
	
	browseScriptsBtn := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err == nil && uri != nil {
				path := uri.Path()
				mw.scriptsPathEntry.SetText(path)
				mw.config.SetScriptsPath(path)
				mw.installer = services.NewInstallerService(mw.config)
				mw.vanillaService = services.NewVanillaService(mw.config.ScriptsPath, path, mw.config.TempPath)
				mw.loadAllMods()
			}
		}, mw.window)
	})
	
	mw.backupCheck = widget.NewCheck("Create backups", func(checked bool) {
		mw.installer = services.NewInstallerService(mw.config)
	})
	mw.backupCheck.SetChecked(false)
	
	mw.modList = widget.NewList(
		func() int { return len(mw.modKeys) },
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			nameLabel := widget.NewLabel("Name")
			descLabel := widget.NewLabel("Description")
			sizeLabel := widget.NewLabel("Size")
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
      if mod.Version == "original" {
        nameLabel.SetText(fmt.Sprintf("%s", mod.Name))
      }

			nameLabel.TextStyle.Bold = true
			descLabel.SetText(mod.Description)
			sizeLabel.SetText(formatFileSize(mod.FileSize))
			
			statusText := ""
			
			// Vérifier le statut selon le type de mod
			if mod.ID == "vanilla_pack" {
				// Mod vanilla pack
				if mw.vanillaService.IsVanillaBacked(&mod) {
					statusText += "💾 Backed up"
				}
			} else {
				// Mod normal
				if mw.downloader.IsModCached(&mod) {
					statusText += "📦 Cached"
				}
			}
			
			if installed, _ := mw.installer.GetInstallationStatus(&mod); installed {
				if statusText != "" { statusText += " | " }
				statusText += "✅ Installed"
			}
			statusLabel.SetText(statusText)
			
			check.SetChecked(mw.selectedMods[modKey])
			check.OnChanged = func(checked bool) {
				mw.toggleModSelection(modKey, checked)
			}
      if mod.FileSize == 0 {
			  check.OnChanged = func(_ bool) {
          check.SetChecked(false);
        }
      }
		},
	)
	
	mw.progressBar = widget.NewProgressBar()
	mw.progressBar.Hide()
	mw.statusLabel = widget.NewLabel("Ready")
	
	mw.installBtn = widget.NewButton("Install selected", mw.installSelectedMods)
	refreshBtn := widget.NewButton("Refresh", mw.refreshModList)
	cacheBtn := widget.NewButton("Cache", mw.showCacheManager)
	
	topSection := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel("Game path:"),
		container.NewBorder(nil, nil, nil, browseGameBtn, mw.gamePathEntry),
		widget.NewLabel("Scripts path:"),
		container.NewBorder(nil, nil, nil, browseScriptsBtn, mw.scriptsPathEntry),
	)
	
	bottomSection := container.NewVBox(
		mw.progressBar,
		mw.statusLabel,
		container.NewHBox(mw.installBtn, refreshBtn, cacheBtn),
	)
	
	modListContainer := container.NewBorder(
		widget.NewLabel("Available mods:"), nil, nil, nil, mw.modList,
	)
	
	lowerSection := container.NewVSplit(modListContainer, bottomSection)
	lowerSection.SetOffset(0.75)
	
	mainContent := container.NewVSplit(topSection, lowerSection)
	mainContent.SetOffset(0.3)
	
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
		mw.statusLabel.SetText("Ready")
	} else {
		mw.statusLabel.SetText(fmt.Sprintf("%d mod(s) selected", count))
	}
}

func (mw *MainWindow) updateGamePathValidation() {
	if mw.installer.IsGamePathValid() {
		mw.statusLabel.SetText("Valid path")
	} else if mw.gamePathEntry.Text != "" {
		mw.statusLabel.SetText("⚠️ Invalid path")
	}
	mw.modList.Refresh()
}

func (mw *MainWindow) installSelectedMods() {
	selectedModKeys := make([]string, 0)
	for key, selected := range mw.selectedMods {
		if selected {
			selectedModKeys = append(selectedModKeys, key)
		}
	}
	
	if len(selectedModKeys) == 0 {
		dialog.ShowInformation("No selection", "Select at least one mod", mw.window)
		return
	}
	
	if !mw.installer.IsGamePathValid() {
		dialog.ShowError(fmt.Errorf("invalid game path (must end with 'Napoleon Total War')"), mw.window)
		return
	}
  if !mw.installer.IsScriptsPathValid() {
		dialog.ShowError(fmt.Errorf("invalid scripts path (must end with 'Napoelon')"), mw.window)
		return
	}
	
	fyne.Do(func() {
		mw.statusLabel.SetText("Preparing...")
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
		
		// Traitement différent pour les mods vanilla
		if mod.ID == "vanilla_pack" {
			fyne.Do(func() {
				mw.statusLabel.SetText(fmt.Sprintf("Restoring %s (%d/%d)", mod.Name, i+1, totalMods))
			})
			
			// Utiliser VanillaService pour restaurer
			err := mw.vanillaService.RestoreVanillaFile(&mod)
			if err != nil {
				fyne.Do(func() {
					mw.statusLabel.SetText(fmt.Sprintf("Restore error %s", mod.Name))
					dialog.ShowError(err, mw.window)
				})
				return
			}
			
			overallProgress := float64(i+1) / float64(totalMods)
			fyne.Do(func() { mw.progressBar.SetValue(overallProgress) })
			continue
		}
		
		// Traitement normal pour les autres mods
		fyne.Do(func() {
			mw.statusLabel.SetText(fmt.Sprintf("Downloading %s (%d/%d)", mod.Name, i+1, totalMods))
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
				mw.statusLabel.SetText(fmt.Sprintf("Download error %s", mod.Name))
				dialog.ShowError(err, mw.window)
			})
			return
		}
		
		fyne.Do(func() {
			mw.statusLabel.SetText(fmt.Sprintf("Installing %s (%d/%d)", mod.Name, i+1, totalMods))
		})
		
		installProgressCallback := func(currentFile string, processed, total int) {
			installProgress := float64(processed) / float64(total)
			overallProgress := (float64(i)*2 + 1 + installProgress) / float64(totalMods*2)
			fyne.Do(func() {
				mw.progressBar.SetValue(overallProgress)
				if currentFile != "" {
					mw.statusLabel.SetText(fmt.Sprintf("Installing %s: %s", mod.Name, currentFile))
				}
			})
		}
		
		err = mw.installer.InstallMod(ctx, &mod, filePath, installProgressCallback)
		if err != nil {
			fyne.Do(func() {
				mw.statusLabel.SetText(fmt.Sprintf("Installation error %s", mod.Name))
				dialog.ShowError(err, mw.window)
			})
			return
		}
		
		overallProgress := float64(i+1) / float64(totalMods)
		fyne.Do(func() { mw.progressBar.SetValue(overallProgress) })
	}
	
	fyne.Do(func() {
		mw.statusLabel.SetText(fmt.Sprintf("Completed (%d mods)", totalMods))
		mw.refreshModList()
		
		_, cacheSize, cacheCount, _ := mw.downloader.GetCacheInfo()
		
		message := fmt.Sprintf("Installation completed!\nMods: %d\nCache: %s (%d files)",
			totalMods, formatFileSize(cacheSize), cacheCount)
		
		dialog.ShowInformation("Completed", message, mw.window)
	})
}

func (mw *MainWindow) refreshModList() {
	availableMods, err := api.FetchAllModMeta()
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}
	
	mw.availableMods = availableMods
	mw.loadAllMods() // Recharger tous les mods incluant vanilla
	
	mw.selectedMods = make(map[string]bool)
	mw.modList.Refresh()
	mw.statusLabel.SetText("Ready")
}

func (mw *MainWindow) showCacheManager() {
	cacheDir, cacheSize, cacheCount, err := mw.downloader.GetCacheInfo()
	if err != nil {
		dialog.ShowError(err, mw.window)
		return
	}
	
	message := fmt.Sprintf("Folder: %s\nSize: %s\nFiles: %d",
		cacheDir, formatFileSize(cacheSize), cacheCount)
	
	dialog.ShowConfirm("Cache", 
		message+"\n\nClear cache?",
		func(confirmed bool) {
			if confirmed {
				if err := mw.downloader.ClearCache(); err != nil {
					dialog.ShowError(err, mw.window)
				} else {
					dialog.ShowInformation("Success", "Cache cleared", mw.window)
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
			Description: "Improves interface", FileSize: 2048576,
			DownloadURL: "https://example.com/mod1.zip",
		},
		"mod2": {
			ID: "mod2", Name: "Extra Content", Version: "2.0.1", 
			Description: "Content mod", FileSize: 10485760,
			DownloadURL: "https://example.com/mod2.zip",
		},
	}
}
