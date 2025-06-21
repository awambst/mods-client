package main

import (
	"log"

	"fyne.io/fyne/v2/app"

	"mod-installer/config"
	"mod-installer/ui"
)

func main() {
	// Initialiser la configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Erreur lors du chargement de la configuration: %v", err)
	}

	// Créer l'application Fyne
	myApp := app.New()
  iconResource := fyne.NewStaticResource("icon.png", iconData)
  myApp.SetIcon(iconResource) // Pas d'icône pour simplifier
	
  // Créer et afficher la fenêtre principale
	mainWindow := ui.NewMainWindow(myApp, cfg)
	mainWindow.ShowAndRun()
}
