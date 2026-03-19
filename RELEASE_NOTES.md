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

**Version actuelle:** 0.1.58
**Dernière mise à jour:** 2026-03-19
