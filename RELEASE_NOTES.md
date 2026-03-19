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

### v0.1.73 (2026-03-19)
- **FIX**: gosec G703 warnings for ProgramData path usage
- **QUALITY**: Added nosec comments with justification for false positive path traversal warnings

### v0.1.72 (2026-03-19)
- **CRITICAL FIX**: Unified config location - service now has PBS config!
- **FIX**: Config, scheduled_jobs, job_history now in C:\ProgramData\NimbusBackup\
- **ROOT CAUSE**: Service had NO PBS config (different UserHomeDir)
- **RESULT**: Backups will now actually reach PBS! 🎯

### v0.1.71 (2026-03-19)
- **FIX**: Unified log location in C:\ProgramData\NimbusBackup\debug.log
- **FEATURE**: GUI and Service now write to same log file (easy debugging!)
- **QUALITY**: No more hidden logs in SYSTEM AppData

### v0.1.70 (2026-03-19)
- **FIX**: Build error - removed unused pbsBackupType variable

### v0.1.69 (2026-03-19)
- **FIX**: Scheduled jobs now use StartBackup (routes via service if available)
- **FIX**: Scheduler no longer bypasses mode detection
- **DEBUG**: Enhanced progress tracking logs (jobID lookup, map size)
- **QUALITY**: Scheduled jobs execute with admin rights when service runs them

### v0.1.68 (2026-03-19)
- **FIX**: Service stop mechanism with stopChan (proper shutdown during upgrades)
- **FIX**: Replace infinite sleep loop with channel-based blocking
- **QUALITY**: Service now stops cleanly on upgrade/uninstall

### v0.1.67 (2026-03-19)
- **FIX**: Lint error S1000 - use for range instead of for-select
- **QUALITY**: Clean code pattern in pollBackupProgress

### v0.1.66 (2026-03-19)
- **FIX**: Service now executes backups in ModeStandalone (VSS with admin rights)
- **FIX**: Service App initialization sets mode explicitly to prevent routing loop
- **FEAT**: Progress polling infrastructure (GUI polls service every 3s)
- **FEAT**: BackupProgress API endpoint GET /backup/status/:jobId
- **FEAT**: Client.GetBackupStatus() for progress queries
- **ARCHITECTURE**: Service executes directly, doesn't route to itself via API

### v0.1.65 (2026-03-19)
- **FIX**: BackupHandler interface type mismatch (GetScheduledJobs signature)
- **FEAT**: Add GetScheduledJobsForAPI() adapter method for API compatibility
- **BUILD**: Compilation now succeeds (type errors resolved)

### v0.1.64 (2026-03-19)
- **FIX**: Service crash on backup (EventsEmit with nil context)
- **FIX**: Protect all EventsEmit calls with ctx nil check
- **FIX**: Initialize service App with context.Background()
- **QUALITY**: Service can now execute backups without crashing

### v0.1.63 (2026-03-19)
- **FEAT**: HTTP API fully integrated (GUI-Service communication)
- **FEAT**: Mode detection (Service vs Standalone) with automatic routing
- **FEAT**: Service exposes HTTP API on localhost:18765
- **FEAT**: GUI detects service and uses API for backups (VSS works!)
- **FEAT**: Fallback to standalone mode if service unavailable
- **FIX**: VSS now works in Service Mode (admin privileges from LocalSystem)
- **FIX**: BackupHandler interface matches actual StartBackup signature

### v0.1.62 (2026-03-19)
- **FIX**: Service now starts after installation (Start="install" Wait="no")
- **FIX**: Service starts automatically after reboot (already had Start="auto")
- **FIX**: Service stops gracefully with StopScheduler() mechanism
- **FEAT**: Scheduler can be stopped with stop channel (prevents hanging shutdown)

### v0.1.61 (2026-03-19)
- **FIX**: Version string mismatch in API server (0.1.58 → 0.1.61)
- **FIX**: golangci-lint errcheck violations (6 issues fixed)
- **FIX**: Remove unused sync.Mutex field from Server struct

### v0.1.60 (2026-03-19)
- **FIX**: Compilation error - use standard log package in API
- **QUALITY**: CI tests now pass (writeDebugLog undefined resolved)

### v0.1.59 (2026-03-19)
- **FEAT**: HTTP API architecture for GUI-Service communication (async backups)
- **FEAT**: Hybrid mode detection (Service/Standalone with auto-fallback)
- **FEAT**: Smart VSS warning (only if !admin AND !service)
- **DOCS**: Complete TODO.md roadmap + multi-server PBS support
- **DOCS**: Enterprise deployment guide (GPO/Intune with config JSON)

### v0.1.58 (2026-03-19)
- **FEAT**: GPL v3 license added to MSI installer (upstream compliance)
- **FIX**: Service no longer starts during installation (prevents hang)
- **BUILD**: MSI installer now completes successfully with proper 64-bit configuration

### v0.1.57 (2026-03-19)
- **FIX**: MSI Platform="x64" declaration for 64-bit components
- **FIX**: Remove custom WiX images to use defaults

### v0.1.56 (2026-03-19) - DEPRECATED
Older versions - see git history

---

**Version actuelle:** 0.1.73
**Dernière mise à jour:** 2026-03-19
