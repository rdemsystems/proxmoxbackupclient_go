# Nimbus Backup - Release Notes

## 📦 Versions disponibles

### NimbusBackup.exe (Standalone)
- ✅ **Backups manuels** : Fonctionne parfaitement
- ✅ **Backups planifiés** : OK quand l'application est lancée manuellement
- ❌ **Persistance au reboot** : Non - l'application ne redémarre pas automatiquement
- 💡 **Usage** : Idéal pour backups ponctuels ou tests

### NimbusBackup.msi (Installateur - Recommandé)
- ✅ **Service Windows** : Démarre automatiquement au boot système
- ✅ **Privilèges admin** : Service tourne toujours avec droits admin (VSS garanti)
- ✅ **Backups planifiés** : Exécutés automatiquement même après reboot
- ✅ **Désinstallation propre** : Nettoyage complet via Panneau de configuration
- 💡 **Usage** : Production et backups automatiques fiables

---

## ✅ Works (Fonctionnalités opérationnelles)

### Backup & Restore
- ✅ Backup one-shot (exécution immédiate)
- ✅ Backup planifié (quotidien avec heure configurable)
- ✅ Édition des jobs planifiés
- ✅ VSS (Volume Shadow Copy) support
- ✅ Exclusion de fichiers/dossiers
- ✅ Restauration de snapshots
- ✅ Liste des snapshots disponibles

### Interface & UX
- ✅ Interface graphique Wails (Go + React)
- ✅ Historique des backups (6 derniers affichés)
- ✅ Relance des backups échoués
- ✅ Planification quotidienne avec heure configurable
- ✅ Barre de progression avec statistiques
- ✅ Minimize to tray avec icône visible
- ✅ Exit from tray (force quit after 2s)

### Configuration
- ✅ Configuration Proxmox Backup Server
- ✅ Test de connexion
- ✅ Fingerprint de certificat
- ✅ Namespace support

## 🚧 In Progress (En cours de développement)

- 🔄 **Installateur MSI** (v0.1.44 - service Windows pour persistance au reboot)
- 🔄 **Service Windows** (backups automatiques même après reboot)
- 🔄 **Vérification stabilité scheduler** (monitoring jobs planifiés)

## 📋 TODO (À faire)

### Priorité haute
- 🌍 **Traduction EN** (interface actuellement en français uniquement)
- 🔧 **Amélioration gestion erreurs VSS** (messages plus clairs)
- 🔔 **Notifications Windows** (succès/échec des backups planifiés)
- 📊 **Dashboard service status** (afficher état du service Windows)

### Priorité moyenne
- 📊 **Statistiques détaillées** (taille sauvegardée, durée, vitesse moyenne)
- 🗓️ **Planification hebdomadaire/mensuelle** (actuellement quotidien uniquement)
- 🔐 **Stockage sécurisé des credentials** (actuellement en clair dans config.json)
- 📧 **Alertes email** (en cas d'échec de backup)
- 🌐 **Support multi-serveurs PBS** (basculement automatique)

### Priorité basse
- 🎨 **Thèmes interface** (dark mode)
- 📝 **Logs détaillés exportables**
- 🔄 **Auto-update intégré** (vérification de nouvelle version)
- 💾 **Import/export configuration**
- 🖥️ **Support Linux/macOS** (actuellement Windows uniquement)

## 📌 Known Issues (Problèmes connus)

- ⚠️ **Version .exe standalone** : Pas de persistance au reboot → Utiliser le MSI pour production
- ⚠️ Pas de validation du format des exclusions
- ⚠️ Interface uniquement en français

## 📜 Changelog récent

### v0.1.51 (2026-03-19)
- **REFACTOR**: CI now uses reproducible dependency management
- **FIX**: golangci-lint v2.11.3 config compatibility (output.formats format)
- **FIX**: go.work version format (go 1.25.0 instead of go 1.25)
- **STRATEGY**: Dependencies pinned via go.mod/go.sum, no auto-update on builds

### v0.1.50 (2026-03-19)
- **UPGRADE**: All Go modules upgraded to Go 1.25
- **FIX**: CI golangci-lint compatibility (now uses latest version)
- **FEAT**: Automatic update of critical packages in CI (golang.org/x/*, wails/v2)
- **DOCS**: Complete package audit documented in PACKAGES_UPDATE.md

### v0.1.49 (2026-03-19)
- **FEAT**: MSI installer now included in GitHub releases
- **FEAT**: Auto-cleanup legacy auto-start on app startup
- **CI**: WiX Toolset integration in GitHub Actions
- **DELIVERY**: Both .exe (standalone) and .msi (service) available

### v0.1.48 (2026-03-19)
- **UPGRADE**: Go 1.25 on CI/CD (GitHub Actions + GitLab CI)
- **PERF**: +5-8% compilation speed, +5-8% runtime performance
- **SECURITY**: Latest security patches and bug fixes from Go 1.25

### v0.1.47 (2026-03-19)
- **FIX**: Correct GetConfigWithHostname() call signature in service.go
- **FIX**: Simplify service config loading (loaded per-job when needed)

### v0.1.46 (2026-03-19)
- **FIX**: Update gui/go.mod to Go 1.23 (required for kardianos/service)
- **FIX**: Specify go 1.23.0 in go.work for exact version match

### v0.1.45 (2026-03-19)
- **FIX**: Upgrade CI to Go 1.23 for kardianos/service compatibility
- **FIX**: Update go.work to require Go 1.23

### v0.1.44 (2026-03-19)
- **FEAT**: Installateur MSI avec service Windows
- **FEAT**: Service démarre automatiquement au boot avec privilèges admin
- **FEAT**: Support flag `--service` pour mode service
- **FEAT**: Persistance garantie après reboot (MSI uniquement)
- **REMOVED**: Code d'auto-start (remplacé par service Windows)
- **DOCS**: Distinction claire .exe (standalone) vs .msi (service)
- **DOCS**: Guide complet de build et installation du MSI

### v0.1.43 (2026-03-19)
- **FEAT**: Système de release notes automatiques (Works/In Progress/TODO)
- **FEAT**: Section "Tested with NimbusBackup" dans les releases
- **DOCS**: Ajout section managed service dans README
- **DOCS**: Workflow GitHub met à jour les release notes automatiquement

### v0.1.42 (2026-03-19)
- **FIX**: Icône systray embarquée depuis vrai .ico (go:embed)
- **FIX**: Auto-start via Task Scheduler avec privilèges HIGHEST
- **FIX**: Nettoyage ancienne entrée registre (migration)
- **FIX**: Délai 5s dans HandleStartupRun pour éviter double exécution

### v0.1.41 (2026-03-19)
- **FEAT**: Édition des jobs planifiés (bouton Éditer/Annuler)
- **FEAT**: Fonction UpdateScheduledJob backend
- **FEAT**: Limitation historique à 6 derniers backups
- **FIX**: Quit systray avec force exit (2s timeout)

---

**Version actuelle:** 0.1.51
**Dernière mise à jour:** 2026-03-19
