{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    # Go et outils de développement
    go
    gopls          # Language server pour Go
    delve          # Debugger Go
    
    # Dépendances système pour Fyne (GUI)
    pkg-config
    xorg.libX11
    xorg.libXcursor
    xorg.libXrandr
    xorg.libXinerama
    xorg.libXi
    xorg.libXext
    xorg.libXxf86vm
    libGL
    libGLU
    
    # Dépendances graphiques communes
    gtk3
    glib
    cairo
    pango
    gdk-pixbuf
    atk
    
    # Outils utiles pour le développement
    git
    curl
    
    # Pour packaging/distribution
    upx  # Compresseur d'exécutables
  ];

  # Variables d'environnement nécessaires
  shellHook = ''
    export CGO_ENABLED=1
    export PKG_CONFIG_PATH="${pkgs.pkg-config}/lib/pkgconfig:${pkgs.gtk3}/lib/pkgconfig"
    
    # Configuration Go
    export GOPATH=$HOME/go
    export PATH=$GOPATH/bin:$PATH
    
    echo "🚀 Environnement Go + Fyne prêt !"
    echo "📁 Initialise ton projet avec: go mod init mod-installer"
    echo "📦 Installe Fyne avec: go get fyne.io/fyne/v2/app fyne.io/fyne/v2/widget"
    echo "🔨 Compile avec: go build ."
    echo "🐧 Cross-compile Linux: GOOS=linux go build -o installer-linux ."
    echo "🪟 Cross-compile Windows: GOOS=windows go build -o installer-windows.exe ."
  '';
}
