# Nimbus Backup - Build Wails

Configuration complète pour builder l'application avec Wails v2.

## Structure du projet

```
gui/
├── main_wails.go         # Point d'entrée avec panic recovery
├── config.go             # Configuration PBS
├── wails.json            # Configuration Wails
├── go.mod                # Dépendances Go
└── frontend/
    ├── package.json      # Dépendances NPM
    ├── vite.config.js    # Configuration Vite
    ├── index.html        # HTML de base
    └── src/
        ├── main.jsx      # Entry point React
        ├── App.jsx       # Application principale
        └── index.css     # Styles
```

## Prérequis

- Go 1.21+
- Node.js 20+
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Windows
- MinGW-w64 (pour CGO)
- WebView2 Runtime (inclus dans Windows 11, installé automatiquement sinon)

## Installation locale

```bash
cd gui

# Installer les dépendances Go
go mod tidy

# Installer les dépendances NPM
cd frontend
npm install
cd ..

# Dev mode
wails dev

# Build production
wails build
```

## Build GitHub Actions

Le workflow `.github/workflows/release.yml` gère automatiquement:

1. ✅ Installation de Go, Node.js, MSYS2 (MinGW)
2. ✅ Installation de Wails CLI
3. ✅ Build avec CGO activé
4. ✅ Embed WebView2 runtime
5. ✅ Compression avec flags `-s -w`
6. ✅ Création de release GitHub

### Déclencher un build

```bash
# Tag et push
git tag v0.5.0
git push origin v0.5.0
```

Le workflow créera automatiquement:
- `NimbusBackup.exe` (artifact)
- `NimbusBackup-Windows.zip` (release)

## Fonctionnalités

### Interface
- ✅ Configuration PBS (API token, datastore, namespace)
- ✅ Import/export JSON config
- ✅ Sauvegarde répertoire ou machine complète
- ✅ Restauration avec listing des snapshots
- ✅ Support VSS (Shadow Copy)

### Robustesse
- ✅ Panic recovery avec stack trace
- ✅ Logs debug dans `~/.proxmox-backup-guardian/debug.log`
- ✅ Gestion d'erreurs complète
- ✅ Console activée pour voir les logs (wails.json: hideConsole: false)

### Build flags
- `-s -w` : Strip symbols, réduit la taille
- `-H=windowsgui` : Mode GUI Windows (pas de console en prod)
- `-trimpath` : Nettoie les paths de build
- `-webview2 embed` : Inclut l'installeur WebView2

## Logs de debug

En cas de crash, consulter:
```
%USERPROFILE%\.proxmox-backup-guardian\debug.log
```

Le log contient:
- Timestamp de chaque étape
- Erreurs avec stack traces
- Appels de fonctions depuis le frontend

## Manifeste Windows

Le fichier `wails.json` inclut:
```json
{
  "windows": {
    "manifest": {
      "level": "requireAdministrator",
      "uiAccess": false
    }
  }
}
```

Cela demande automatiquement les droits admin au lancement (nécessaire pour VSS).

## Taille du binaire

Build optimisé attendu: **15-20 MB**
- Inclut WebView2 installer (~7 MB)
- Runtime Wails (~5 MB)
- Application + frontend (~3-5 MB)

Si le binaire dépasse 30 MB, vérifier que `-s -w -trimpath` est bien utilisé.

## Troubleshooting

### Crash au lancement
1. Vérifier `debug.log`
2. Lancer depuis cmd.exe pour voir stderr
3. Vérifier que WebView2 est installé

### Build échoue (CGO)
1. Vérifier MSYS2 installé
2. Vérifier `CGO_ENABLED=1`
3. Ajouter MinGW au PATH: `C:\msys64\mingw64\bin`

### Frontend ne se charge pas
1. Vérifier `npm run build` fonctionne
2. Vérifier que `frontend/dist` existe
3. Vérifier `go:embed all:frontend/dist` dans main.go

## Contact

RDEM Systems - https://nimbus.rdem-systems.com
