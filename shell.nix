{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    # Go et outils de développement
    go
    gopls          # Language server pour Go
    delve          # Debugger Go
    
    # Cross-compilation pour Windows
    #pkgsCross.mingwW64.stdenv.cc
    #pkgsCross.mingwW64.windows.mingw_w64_pthreads
    
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
    
    # Cross-compilation Windows
    export CC_FOR_TARGET_windows_amd64="${pkgs.pkgsCross.mingwW64.stdenv.cc}/bin/x86_64-w64-mingw32-gcc"
    
    echo "🚀 Environnement Go + Fyne prêt !"
    echo "📁 Initialise ton projet avec: go mod init mod-installer"
    echo "📦 Installe Fyne avec: go get fyne.io/fyne/v2/app fyne.io/fyne/v2/widget"
    echo "🔨 Compile Linux: go build -o mod-installer ."
    echo "🪟 Cross-compile Windows: CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -o mod-installer.exe ."
    echo "📦 Ou utilise fyne package: fyne package -os windows -o mod-installer.exe"
  '';
}
