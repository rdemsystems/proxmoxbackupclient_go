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
- ✅ **Auto-split pour backups >100GB** (découpage intelligent en jobs de ~100GB)
- ✅ Édition des jobs planifiés
- ✅ VSS (Volume Shadow Copy) support
- ✅ Exclusion de fichiers/dossiers
- ✅ Restauration de snapshots
- ✅ Liste des snapshots disponibles
- ✅ Backups longue durée robustes (keep-alive 30s)

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

- 🔄 **Architecture binaire séparée** - ✅ Complété en v0.2.0
- 🔄 **Auto-split backups** - ✅ Complété en v0.2.1
- 🔄 **Tests en production** (validation backups 11h+ avec keep-alive 30s)

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

### v0.2.12 (2026-03-23)
- **FIX**: Better responsive design for very small screens (400x300 min, adaptive text)
- **UX**: Works on VM displays and low-resolution screens (800x600, Proxmox VE console)

### v0.2.11 (2026-03-23)
- **FIX**: UI overflow on small screens - Added minimum window size (600x500)
- **UX**: Better usability on low-resolution displays

### v0.2.10 (2026-03-23)
- **FIX**: MSI build - Incorrect service exe path

### v0.2.9 (2026-03-23)
- **FIX**: Service build - app.App missing BackupHandler interface methods

### v0.2.8 (2026-03-23)
- **FIX**: GUI build error - Missing gui/api in go.mod


### v0.2.7 (2026-03-23)
- **FIX**: Missing gui/api/go.mod module definition

### v0.2.6 (2026-03-23)
- **FIX**: Service build error - "gui is a program, not an importable package"
- **REFACTOR**: Extracted App to gui/app package (importable)
- **BUILD**: Service now builds successfully in CI/CD

### v0.2.5 (2026-03-23)
- Re-release of v0.2.4 with proper build sequence (no functional changes)

### v0.2.4 (2026-03-23)
- **FIX**: Service build error in CI/CD - Missing replace directives
- **FIX**: Malformed module path errors for local modules resolved
- **BUILD**: cmd/service/go.mod now properly resolves all local dependencies

### v0.2.3 (2026-03-23) - Complete Release Since v0.2.0

#### 🎯 Major Features (v0.2.0 → v0.2.3)

**Binary Separation Architecture (v0.2.0)**
- **BREAKING**: Separate GUI and Service executables
  - `NimbusBackup.exe` - GUI application (Wails v2)
  - `NimbusBackupSVC.exe` - Windows Service (kardianos/service)
- **HTTP API** on localhost:18765 for GUI-Service communication
- **Single instance enforcement** with Windows mutex
- **MSI installer** with dual binary support and automatic service installation

**Long Backup Reliability (v0.2.0)**
- **CRITICAL FIX**: Keep-alive interval changed from 5 minutes to 30 seconds
- Prevents "dynamic writer not registered" HTTP/2 errors after 11+ hour backups
- Maintains TCP connection during local file processing pauses (chunking, hashing)
- Fixes both client timeout (~50s) and firewall timeout (~60s)

**Auto-Split for Large Backups (v0.2.1)**
- Intelligent job splitting for backups >100GB threshold
- Bin-packing algorithm distributes folders into balanced jobs (~100GB each)
- Confirmation dialog shows total size and suggested split count
- Sequential execution with per-job retry capability
- **Benefit**: If one job fails, only retry ~100GB instead of losing 11+ hours
- Max 10 jobs per backup to prevent over-fragmentation

**Smart System Exclusions (v0.2.2 & v0.2.3)**
- **Auto-exclusion of Windows system folders**:
  - `System Volume Information` - VSS snapshots storage (can be 100+ GB)
  - `$RECYCLE.BIN` - Windows recycle bin
  - `Recovery` - Windows recovery partition data
- **Auto-exclusion of Windows system files**:
  - `pagefile.sys`, `hiberfil.sys`, `swapfile.sys`
  - `DumpStack.log.tmp` (crash dump temporary)
- **Impact**: Backup size matches actual files instead of including hidden system data
- **Example**: Drive shows 1.03 TB used but files are 141 GB → backup will be ~141 GB

#### 🐛 Bug Fixes

**v0.2.3**
- Auto-exclude Windows system folders/files in file-mode backups

**v0.2.2**
- CI/CD build error - Service executable not built before MSI creation
- LGHT0103 error resolved (missing NimbusBackupSVC.exe)

**v0.2.0**
- MSI upgrade schedule fixed (afterInstallInitialize)
- Service stop mechanism during upgrades
- Kill GUI processes before upgrade to prevent hang

#### 🏗️ Architecture Changes

**HTTP API Endpoints** (v0.2.0)
- `/health` - Service status check
- `/config` - Get/update configuration
- `/jobs` - List scheduled jobs
- `/jobs/create`, `/jobs/update`, `/jobs/delete/{id}` - Job management
- `/backup/start` - Execute backup immediately

**Build System** (v0.2.2)
- GitHub Actions workflow builds service before MSI packaging
- Both binaries (GUI + Service) copied to dist/ folder

#### 📊 Statistics & Examples

**Before v0.2.0:**
- 11-hour backup fails after 52 seconds of local processing
- Error: "dynamic writer '1' not registered HTTP/2.0"

**After v0.2.0:**
- Keep-alive every 30 seconds prevents timeout
- 11+ hour backups complete successfully

**Before v0.2.1:**
- 864 GB backup = single job
- If it fails at 99%, lose 11 hours of work

**After v0.2.1:**
- 864 GB backup = 9 jobs of ~96 GB each
- If one fails, only retry that ~100 GB chunk

**Before v0.2.3:**
- Backing up `D:\` includes 890 GB of VSS snapshots
- Total backup: 1.03 TB (141 GB files + 890 GB snapshots)

**After v0.2.3:**
- VSS snapshots auto-excluded
- Total backup: ~141 GB (actual files only)

#### 🔧 Migration Notes

**Upgrading from v0.1.x to v0.2.x:**
- MSI installer handles upgrade automatically
- Old single binary replaced with two executables
- Service automatically stopped, upgraded, restarted
- Configuration preserved in `%ProgramData%\NimbusBackup\`
- No user action required

#### 📝 Recommendations

**Backup Strategy:**
- **File-level backups** (default): Use file mode with auto-exclusions
  - Excludes VSS snapshots, recycle bin, paging files
  - Faster, smaller, suitable for file/folder restore
- **Bare-metal restore**: Use disk mode in separate job
  - Includes everything (boot sector, system files, etc.)
  - Suitable for full system recovery

**Large Backups (>100 GB):**
- Accept auto-split suggestion when prompted
- Each job ~100 GB max for better reliability
- Per-job retry instead of all-or-nothing

---

### v0.2.2 (2026-03-23)
- **FEATURE**: Auto-exclusion of Windows system folders (System Volume Information, $RECYCLE.BIN, Recovery)
- **FEATURE**: Auto-exclusion of Windows system files (pagefile.sys, hiberfil.sys, swapfile.sys)
- **IMPORTANT**: File-mode backups now skip VSS snapshots storage (can save 100s of GB)
- **FIX**: CI/CD build error - Service executable not built before MSI creation
- **FIX**: LGHT0103 error resolved (missing NimbusBackupSVC.exe)
- **BUILD**: GitHub Actions workflow now builds service before MSI packaging

### v0.2.1 (2026-03-23)
- **FEATURE**: Auto-split for large backups (>100GB threshold)
- **FEATURE**: Bin-packing algorithm distributes folders into balanced jobs
- **UX**: Confirmation dialog shows total size and suggested split count
- **ARCHITECTURE**: Sequential execution with per-job retry capability
- **BENEFIT**: If one job fails, only retry ~100GB instead of losing 11+ hours

### v0.2.0 (2026-03-23)
- **BREAKING**: Binary separation architecture (GUI + Service)
- **ARCHITECTURE**: NimbusBackup.exe (GUI) + NimbusBackupSVC.exe (Service)
- **FEATURE**: HTTP API on localhost:18765 for GUI-Service communication
- **FEATURE**: Single instance enforcement (Windows mutex)
- **CRITICAL FIX**: Keep-alive interval changed from 5min to 30s
- **FIX**: Prevents "dynamic writer not registered" errors after 11+ hour backups
- **FIX**: MSI installer dual binary support

### v0.1.92 (2026-03-21)
- **FIX**: MSI upgrade now uses afterInstallInitialize schedule
- **FIX**: Remove duplicate ServiceControl elements
- **IMPROVEMENT**: AllowSameVersionUpgrades for easier testing
- **QUALITY**: Service should stop cleanly before upgrade

### v0.1.91 (2026-03-21)
- **FIX**: Build error - missing totalSize variable declaration
- **QUALITY**: CI tests now pass

### v0.1.90 (2026-03-21)
- **FEATURE**: Backup completion report shows duration and size
- **UX**: "Backup completed in 2m 34s: 1.8 GB backed up (89 new, 565 reused chunks)"
- **FIX**: MSI upgrade now stops service before installing (Wait=yes)
- **QUALITY**: Human-readable duration format (Xm Ys or Xh Ym)

### v0.1.89 (2026-03-21)
- **CRITICAL FIX**: Service scheduled jobs blocked on Wails EventsEmit
- **ROOT CAUSE**: context.Background() is not nil, triggered Wails event emission
- **FIX**: Service process never emits Wails events (no runtime available)
- **RESULT**: Scheduled backups now execute correctly via service

### v0.1.88 (2026-03-20)
- **DEBUG**: Trace logs to identify backup hang point
- **DEBUG**: Logs after progress(0.05), client creation, directory loop
- **TROUBLESHOOTING**: Find where backup blocks after "Connecting to PBS"

### v0.1.87 (2026-03-20)
- **CRITICAL FIX**: Infinite loop when service re-detects itself as available
- **FIX**: Service process never re-detects mode (stays in Standalone)
- **ROOT CAUSE**: Mode re-detection in v0.1.84 caused service to route to itself
- **RESULT**: Backups now execute correctly without recursive API calls

### v0.1.86 (2026-03-20)
- **DEBUG**: Enhanced scheduler logging (job count, NextRun, ShouldRun)
- **DEBUG**: Logs when no jobs found or jobs disabled
- **QUALITY**: Easier troubleshooting for scheduled backups not running

### v0.1.85 (2026-03-20)
- **FIX**: Separate log files to avoid concurrent write issues
- **ARCHITECTURE**: debug-gui.log for GUI, debug-service.log for Service
- **BENEFIT**: No file locking conflicts, cleaner debugging per process

### v0.1.84 (2026-03-20)
- **CRITICAL FIX**: Mode re-detection on each backup (fixes missing progress bar)
- **FIX**: GUI now switches to Service mode if service becomes available after startup
- **ROOT CAUSE**: Mode detected once at startup, never re-checked if service started late
- **RESULT**: Progress bar now displays during backup execution via service

### v0.1.83 (2026-03-20)
- **CRITICAL FIX**: Progress callbacks now use map with mutex (fixes race condition)
- **FIX**: Concurrent backups no longer overwrite each other's progress callbacks
- **ARCHITECTURE**: Callbacks stored per jobID, supports multiple simultaneous backups
- **DEBUG**: Enhanced logging for callback registration and execution flow
- **QUALITY**: Service progress updates now reliably reach GUI frontend

### v0.1.82 (2026-03-20)
- **CRITICAL FIX**: Test Connection now performs real HTTP call to PBS
- **FIX**: Detects DNS typos immediately (was showing OK with wrong hostname)
- **FIX**: Clear error messages: connection failed, auth failed, access denied
- **SECURITY**: 10s timeout prevents hanging on unreachable servers

### v0.1.81 (2026-03-20)
- **CRITICAL FIX**: Service now reloads config before each backup (no restart needed)
- **FIX**: Pre-fill backup dirs from last successful backup (UX improvement)
- **ARCHITECTURE**: Multi-PBS support planned (pbs1, pbs2, etc.)
- **ARCHITECTURE**: Jobs managed by service, editable via API
- **TODO**: Added API endpoints for reload config/jobs (`POST /api/reload/config`)
- **TODO**: Remote API for MSP provisioning documented

### v0.1.80 (2026-03-20)
- **FIX**: backup-id now falls back to hostname in SaveConfig & TestConnection
- **FIX**: VSS warning only shows when mode=Standalone AND !is_admin
- **FEATURE**: Smart VSS warnings - info message when service available
- **UX**: No more misleading VSS admin warnings when service has privileges
- **UX**: Importing minimal JSON config no longer creates empty fields

### v0.1.79 (2026-03-19)
- **CRITICAL FIX**: VSS admin check now only in standalone mode (not when using service)
- **FIX**: Service can now use VSS without GUI being admin
- **FEATURE**: GetSystemInfo() API for mode/admin status detection
- **FEATURE**: DiagnoseConfig() API for debugging config issues
- **DEBUG**: Enhanced logging in SaveConfig to track save failures

### v0.1.78 (2026-03-19)
- **CRITICAL FIX**: Service logs accessible even if %ProgramData% env var missing
- **FIX**: Hardcoded fallback to C:\ProgramData\NimbusBackup on Windows
- **FIX**: Prevents service logs from being written to SYSTEM profile directory
- **QUALITY**: Service logs guaranteed to be in accessible location

### v0.1.77 (2026-03-19)
- **FIX**: Empty backup-id now fallbacks to hostname (as intended)
- **FIX**: Backup no longer fails with empty backup-id field
- **DOC**: Clarify service logs location (C:\ProgramData\NimbusBackup\debug.log)

### v0.1.76 (2026-03-19)
- **UX**: Test Connection now tests form values without saving first
- **FEATURE**: TestConnection() accepts optional config parameter
- **IMPROVEMENT**: Users can test configuration before committing to save

### v0.1.75 (2026-03-19)
- **FIX**: Safe trim() with fallback for undefined config fields (JSON import crash fix)
- **QUALITY**: Frontend now handles incomplete/partial config JSON gracefully

### v0.1.74 (2026-03-19)
- **FEATURE**: Real-time backup progress tracking via API callbacks
- **FEATURE**: Custom progress callbacks for API server mode
- **FIX**: Remove unused getJobHistoryPathLegacy function
- **ARCHITECTURE**: Progress updates now flow from backup execution to API progress map

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

**Version actuelle:** 0.2.12
**Dernière mise à jour:** 2026-03-23
