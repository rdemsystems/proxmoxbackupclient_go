# Proxmox Backup Guardian - GUI

Interface graphique native pour le client Proxmox Backup.

## 📦 Deux versions disponibles :

### 1. **CLI** (Léger - ~10-20 MB)
```bash
./build.sh
./directorybackup -baseurl ... -authid ...
```

✅ Léger et rapide
✅ Parfait pour scripts et serveurs
✅ Pas de dépendances graphiques

### 2. **GUI** (Plus lourd - ~30-50 MB)
```bash
./build_gui.sh
./proxmox-backup-gui
```

✅ Interface utilisateur intuitive
✅ Configuration visuelle
✅ Monitoring en temps réel
✅ Pas besoin de ligne de commande

## 🚀 Installation

### Linux

```bash
cd /home/rdem/git/proxmoxbackupclient_go

# Installer Fyne
go install fyne.io/fyne/v2/cmd/fyne@latest

# Build GUI
chmod +x build_gui.sh
./build_gui.sh

# Lancer
./proxmox-backup-gui
```

### Windows

```cmd
cd C:\path\to\proxmoxbackupclient_go

REM Build GUI
build_gui.bat

REM Lancer
proxmox-backup-gui.exe
```

### macOS

```bash
./build_gui.sh

# Créer un bundle .app
fyne package -os darwin -icon Icon.png
```

## 🎨 Fonctionnalités GUI

### Onglet Configuration PBS
- URL du serveur PBS
- Empreinte certificat SSL (SHA-256)
- Authentication ID (API Token)
- Secret
- Datastore et Namespace
- **Test de connexion** intégré
- **Sauvegarde** de la configuration

### Onglet Sauvegarde
- **Sélection du répertoire** à sauvegarder (avec parcourir)
- Backup ID personnalisé
- Option VSS (Windows)
- **Barre de progression** en temps réel
- **Statut** détaillé
- Boutons Start/Stop

### Onglet À propos
- Version de l'application
- Informations projet
- Lien vers documentation

## 📁 Configuration

La config GUI est stockée dans :
- **Linux/macOS** : `~/.proxmox-backup-guardian/config.json`
- **Windows** : `%USERPROFILE%\.proxmox-backup-guardian\config.json`

Format :
```json
{
  "baseurl": "https://pbs.example.com:8007",
  "certfingerprint": "AA:BB:CC:...",
  "authid": "backup@pbs!token",
  "secret": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "datastore": "backup-prod",
  "namespace": "production",
  "backupdir": "/data",
  "backup-id": "server-01",
  "usevss": true
}
```

## 🔧 Architecture

```
gui/
├── main.go       # Interface Fyne + logic UI
├── config.go     # Gestion configuration
├── backup.go     # Exécution backups (appelle CLI)
└── go.mod        # Dépendances Fyne
```

### Intégration avec CLI

La GUI **appelle le binaire CLI** (`directorybackup`) en sous-processus :
- Parse la sortie stdout pour progression
- Extrait les stats (chunks, speed, etc.)
- Met à jour l'UI en temps réel

## 📊 Monitoring temps réel

Le GUI parse la sortie du CLI pour extraire :
- **Progression** : `Progress: 67%`
- **Vitesse** : `Speed: 45.2 MB/s`
- **Chunks** : `Chunks: 1245 new, 8932 reused`
- **Statut** : `Uploading`, `Processing`, etc.

## 🎯 TODO

- [x] Interface configuration PBS
- [x] Sélection répertoire backup
- [x] Barre de progression
- [ ] **Test connexion PBS réel** (appeler API PBS)
- [ ] **Exécution backup réel** (intégrer directorybackup)
- [ ] Parse stdout pour stats réelles
- [ ] **Icône systray** (minimiser dans tray)
- [ ] **Notifications desktop** (fin de backup)
- [ ] **Planification** (cron/scheduled tasks)
- [ ] **Logs** détaillés dans l'UI
- [ ] **Historique** des backups
- [ ] **Restauration** via GUI
- [ ] **Multi-configs** (plusieurs serveurs PBS)
- [ ] Thème dark/light
- [ ] Traductions (FR/EN/DE)

## 🔐 Sécurité

- Config stockée avec permissions `0600` (lecture seule owner)
- Secret affiché comme mot de passe (masqué)
- Pas de transmission réseau (appelle CLI local)

## 📝 Distribution

### Créer un installateur

**Windows (NSIS):**
```cmd
fyne package -os windows -icon Icon.png
```

**Linux (AppImage/Flatpak):**
```bash
fyne package -os linux -icon Icon.png
```

**macOS (.app bundle):**
```bash
fyne package -os darwin -icon Icon.png
```

## 💡 Utilisation recommandée

**Pour utilisateurs desktop** → GUI
**Pour serveurs/scripts** → CLI
**Pour automatisation** → CLI + cron/scheduled tasks

## 🆚 Comparaison binaires

| Feature | CLI | GUI |
|---------|-----|-----|
| Taille | ~10-20 MB | ~30-50 MB |
| Dépendances | Aucune | Fyne, libs graphiques |
| Automatisation | ✅ Excellent | ⚠️ Manuel |
| Interface | Terminal | Graphique native |
| Serveur headless | ✅ Parfait | ❌ Nécessite X11 |
| Utilisateur lambda | ⚠️ Complexe | ✅ Simple |

## 📞 Support

- GitHub : https://github.com/tizbac/proxmoxbackupclient_go
- Site web : https://nimbus.rdem-systems.com
- Email : contact@rdem-systems.com

---

**Développé avec Fyne pour Go** 🎨
