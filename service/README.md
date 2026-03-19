# Nimbus Backup Service

Service en arrière-plan pour exécuter les backups programmés automatiquement.

## Installation

### Windows

```powershell
# Installer le service (avec privilèges administrateur)
nimbus-backup-service.exe -service install

# Démarrer le service
nimbus-backup-service.exe -service start

# Vérifier le statut
sc query NimbusBackupService
```

### Linux (systemd)

```bash
# Installer le service
sudo ./nimbus-backup-service -service install

# Démarrer le service
sudo systemctl start NimbusBackupService

# Activer au démarrage
sudo systemctl enable NimbusBackupService

# Vérifier le statut
sudo systemctl status NimbusBackupService
```

## Configuration

Le service utilise les mêmes fichiers de configuration que la GUI:
- **Config PBS**: `~/.proxmox-backup-guardian/config.json`
- **Jobs programmés**: `~/.proxmox-backup-guardian/jobs.json`

Configurez vos backups via la GUI, le service les exécutera automatiquement.

## Commandes

```bash
# Installer
nimbus-backup-service -service install

# Désinstaller
nimbus-backup-service -service uninstall

# Démarrer
nimbus-backup-service -service start

# Arrêter
nimbus-backup-service -service stop

# Redémarrer
nimbus-backup-service -service restart

# Exécuter en mode interactif (debug)
nimbus-backup-service
```

## Logs

### Windows
Logs dans l'Event Viewer: `Applications and Services Logs → NimbusBackupService`

### Linux
Logs via journalctl:
```bash
sudo journalctl -u NimbusBackupService -f
```

## Architecture

Le service:
1. Tourne en continu en arrière-plan
2. Vérifie toutes les minutes s'il y a des jobs à exécuter
3. Exécute les backups programmés automatiquement
4. Met à jour les timestamps de dernière/prochaine exécution
5. Log toutes les opérations

## TODO

- [ ] Implémenter exécution de backup réelle (pbscommon)
- [ ] Parser cron schedules correctement (github.com/robfig/cron)
- [ ] Notifications email en cas d'échec
- [ ] Métriques et monitoring
- [ ] Interface web de monitoring (optionnel)
