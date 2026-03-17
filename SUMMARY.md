# 🎨 Proxmox Backup Guardian - GUI Client

## ✅ Résumé de ce qui a été créé

### 📁 Structure du projet

```
proxmoxbackupclient_go/
├── gui/                              # ⭐ NOUVEAU: Interface graphique
│   ├── main.go                       # Point d'entrée, interface Fyne
│   ├── config.go                     # Gestion configuration PBS
│   ├── backup.go                     # Exécution backups (appelle CLI)
│   ├── backup_config_ui.go           # UI avancée (multi-folders, exclusions, etc.)
│   ├── jobs.go                       # Gestion des jobs de backup
│   ├── jobs_ui.go                    # Interface liste des jobs
│   ├── scheduler.go                  # Planification (cron/Task Scheduler)
│   └── go.mod                        # Dépendances Fyne
│
├── build_gui.sh                      # ⭐ Script build GUI (Linux/macOS)
├── build_gui.bat                     # ⭐ Script build GUI (Windows)
│
├── GUI_README.md                     # ⭐ Documentation GUI
├── CONFIG_FORMAT_SPEC.md             # ⭐ Format JSON/INI pour members.rdem-systems.com
├── INTEGRATION_MEMBERS.md            # ⭐ Guide intégration Laravel
└── SUMMARY.md                        # ⭐ Ce fichier
```

### 🎯 Fonctionnalités implémentées

#### 1. Interface graphique native (Fyne)
- ✅ Configuration PBS visuelle
- ✅ Sélection multi-dossiers
- ✅ Détection et sélection de disques
- ✅ Exclusions avec presets (dev, temp, caches, media)
- ✅ Planification automatique (cron/Task Scheduler)
- ✅ Politique de rétention configurable
- ✅ Options avancées (compression, chunk size, bande passante)

#### 2. Gestion des jobs
- ✅ Création/modification/suppression de jobs
- ✅ Sauvegarde en JSON (format structuré)
- ✅ Export en INI (compatible CLI)
- ✅ Liste des jobs avec statuts
- ✅ Activation/désactivation par job

#### 3. Planification automatique
- ✅ **Linux/macOS** : intégration crontab
- ✅ **Windows** : intégration Task Scheduler
- ✅ Presets : horaire, 6h, quotidien, hebdo, mensuel
- ✅ Cron personnalisé

#### 4. Configuration
- ✅ Stockage sécurisé : `~/.proxmox-backup-guardian/`
  - `config.json` - Config PBS globale
  - `jobs.json` - Liste des jobs
- ✅ Permissions 0600 (lecture seule owner)
- ✅ Import/Export JSON et INI

## 🚀 Utilisation

### Build

```bash
# GUI (interface graphique - ~40 MB)
./build_gui.sh

# CLI (ligne de commande - ~10 MB)
./build.sh
```

### Lancer la GUI

```bash
# Linux/macOS
./proxmox-backup-gui

# Windows
proxmox-backup-gui.exe
```

### CLI traditionnel

```bash
# Toujours disponible !
./directorybackup -baseurl https://pbs:8007 -authid user@pbs!token -secret xxx -datastore backup -backupdir /data
```

## 📦 Deux binaires distincts

| | CLI | GUI |
|---|-----|-----|
| **Taille** | ~10-20 MB | ~30-50 MB |
| **Dépendances** | Aucune | Fyne + libs graphiques |
| **Usage** | Scripts, serveurs | Desktop, config visuelle |
| **Automatisation** | ✅ Excellent | ⚠️ Manuel |
| **Facilité** | ⚠️ Commande complexe | ✅ Interface intuitive |
| **Headless** | ✅ Parfait | ❌ Nécessite X11/GUI |

## 🔗 Intégration members.rdem-systems.com

### Workflow

```
Client achète offre → RDEM configure PBS → Client télécharge:
                                             1. proxmox-backup-gui.exe
                                             2. backup-config.json (pré-rempli)
                                          → Client importe JSON dans GUI
                                          → ✅ Config PBS automatique
                                          → Client configure dossiers
                                          → Active planification
```

### Fichiers Laravel à créer

Voir **INTEGRATION_MEMBERS.md** pour code complet :

1. **Migration** : `backup_subscriptions` table
2. **Modèle** : `BackupSubscription.php`
3. **Contrôleur** : `BackupController.php`
4. **Routes** : `/services/backup/*`
5. **Vue** : `resources/views/services/backup.blade.php`

### Exemple JSON généré

```json
{
  "pbs_config": {
    "baseurl": "https://pbs-fr-paris.rdem-systems.com:8007",
    "authid": "acme-corp@pbs!backup-production",
    "secret": "a1b2c3d4-...",
    "datastore": "acme-corp-2tb",
    "backup-id": "acme-corp"
  },
  "schedule_cron": "0 2 * * *",
  "keep_daily": 30
}
```

## 🎨 Captures d'écran de l'interface

### Onglet Configuration PBS
- URL serveur, certificat SSL
- Authentication ID, Secret (masqué)
- Datastore, Namespace
- **Bouton "Tester"** pour vérifier connexion

### Onglet Sauvegarde → Source
- **Multi-sélection de dossiers**
  - Bouton "Ajouter un dossier"
  - Liste avec icônes
  - Retrait facile
- **Détection de disques** (machine backup)
  - PhysicalDisk0, PhysicalDisk1, etc.
  - Affichage taille et modèle

### Onglet Sauvegarde → Exclusions
- Liste d'exclusions
- Champ ajout : `*.tmp, node_modules/`
- **Presets** :
  - Dev (node_modules, .git, dist)
  - Temp files (*.tmp, ~*, .DS_Store)
  - Caches (__pycache__, .cache)
  - Media (*.mp4, *.mkv)

### Onglet Sauvegarde → Planification
- ☑️ Activer planification
- Select: Horaire, 6h, Quotidien, Hebdo, Mensuel, Personnalisé
- Input cron personnalisé : `0 2 * * *`
- Lien aide crontab.guru

### Onglet Sauvegarde → Rétention
- Keep last: 7
- Keep daily: 14
- Keep weekly: 8
- Keep monthly: 12
- Explication politique de rétention

### Onglet Sauvegarde → Avancé
- Compression: zstd / lz4 / none
- Chunk size: 4MB / 2MB / 8MB / 16MB
- Limite bande passante (MB/s)
- Uploads parallèles

### Onglet Jobs
- Liste des jobs avec statuts ✅/❌
- Dernière exécution
- Boutons : Nouveau, Éditer, Lancer, Supprimer
- Export JSON/INI par job

## 📋 TODO / Améliorations futures

### Court terme
- [ ] Implémenter **test connexion PBS réel** (appel API)
- [ ] **Parser stdout** du CLI pour stats temps réel
- [ ] **Logs** détaillés visibles dans GUI
- [ ] **Systray icon** (minimiser dans barre)
- [ ] **Notifications desktop** (fin backup)

### Moyen terme
- [ ] **Historique** des backups dans GUI
- [ ] **Restauration** guidée avec browser
- [ ] **Multi-serveurs PBS** (switch entre comptes)
- [ ] **Graphiques** de tendances (espace utilisé, dédup)
- [ ] Thème **dark mode**
- [ ] **Traductions** FR/EN/DE/ES

### Long terme
- [ ] **Dashboard web** embarqué (API REST locale)
- [ ] **Mobile app** pour monitoring
- [ ] **Chiffrement client-side** (quand implémenté dans CLI Go)
- [ ] **Cloud sync** des configs (optionnel)

## 🎓 Documentation

- **GUI_README.md** : Guide complet GUI
- **CONFIG_FORMAT_SPEC.md** : Format JSON/INI, exemples Laravel
- **INTEGRATION_MEMBERS.md** : Intégration site members
- **README.md** (principal) : Client Go original

## 🔐 Sécurité

- ✅ Config stockée avec permissions `0600`
- ✅ Secret PBS affiché comme password (masqué)
- ✅ Aucune transmission réseau (sauf vers PBS)
- ✅ Appelle CLI local (pas de code distant)
- ⚠️ TODO: Chiffrement des credentials en local

## 📞 Support & Contribution

- **Client Go original** : https://github.com/tizbac/proxmoxbackupclient_go
- **Site commercial** : https://nimbus.rdem-systems.com
- **Members portal** : https://members.rdem-systems.com
- **Email** : contact@rdem-systems.com

---

## ⚡ Quick Start

```bash
# 1. Build
cd /home/rdem/git/proxmoxbackupclient_go
./build_gui.sh

# 2. Lancer
./proxmox-backup-gui

# 3. Importer config depuis members.rdem-systems.com
#    Fichier → Importer → backup-acme-corp.json

# 4. Configurer dossiers
#    Onglet Sauvegarde → Source → Ajouter un dossier

# 5. Activer planification
#    Onglet Planification → ☑️ Activer → Quotidien

# 6. Créer le job
#    Jobs → Nouveau Job → ✅ Créé et planifié
```

## 🎉 Résultat final

**Vous avez maintenant :**
- ✅ Une GUI native professionnelle (Fyne)
- ✅ Gestion complète des backups PBS
- ✅ Planification automatique système
- ✅ Multi-dossiers, exclusions, rétention
- ✅ Export JSON/INI
- ✅ Intégration members.rdem-systems.com
- ✅ 2 binaires : CLI léger + GUI complet

**Votre workflow de vente :**
1. Client achète sur vault-backup-guardian
2. RDEM configure PBS server-side
3. Client télécharge GUI + config JSON pré-remplie
4. Client importe, configure, active
5. Backups automatiques ! 🎊

---

**Développé avec ❤️ pour RDEM Systems & la communauté Proxmox Backup**
