# Nimbus Backup - TODO

## 🔴 P0 - CRITIQUE (En cours)

### Communication GUI ↔ Service (HTTP Local)
**Choix: HTTP sur localhost - dev rapide + évolution entreprise**

- [ ] **HTTP Server** (Service Windows)
  - [ ] Créer `gui/api/server.go`
  - [ ] Port: `localhost:18765` (bind 127.0.0.1 uniquement)
  - [ ] Endpoints REST:
    - `POST /backup` - Lance un backup
    - `GET /status` - État du service
    - `GET /jobs` - Liste des jobs configurés
    - `DELETE /job/{id}` - Cancel job en cours
  - [ ] Auth: Token simple (généré au install, stocké dans config)
  - [ ] Timeout: 30s par requête
  - [ ] CORS: disabled (localhost only)
  - [ ] Error handling: JSON errors + logs

- [ ] **HTTP Client** (GUI Wails)
  - [ ] Créer `gui/api/client.go`
  - [ ] Wrapper: `client.StartBackup()`, `client.GetStatus()`
  - [ ] Retry logic: 3 tentatives si service occupé
  - [ ] Fallback: afficher erreur propre si service down
  - [ ] UI: spinner pendant l'attente de réponse

- [ ] **Refactoring GUI**
  - [ ] Remplacer appels directs backup par `pipe_client.SendBackupCommand()`
  - [ ] Bouton "Backup" → appel pipe au lieu d'exec direct
  - [ ] Status bar: poll `GetServiceStatus()` toutes les 5s

- [ ] **Tests**
  - [ ] Service seul sans GUI → fonctionne
  - [ ] GUI seul sans Service → erreur propre
  - [ ] Backup VSS via HTTP → succès
  - [ ] 2 GUI simultanés → pas de race condition
  - [ ] Firewall Windows: autoriser localhost (pas de prompt)

**Temps estimé:** 3-4 heures (HTTP plus rapide que Named Pipes!)

---

## 🟠 P1 - IMPORTANT (Prochaines semaines)

### Service Windows - Robustesse
- [ ] **VSS Cleanup au démarrage**
  ```go
  func cleanupOrphanedVSS() {
      exec.Command("vssadmin", "delete", "shadows", "/for=C:", "/all", "/quiet").Run()
  }
  ```
  - [ ] Appel dans `service.Start()`
  - [ ] Log les shadows supprimées

- [ ] **Working Directory fix**
  ```go
  exePath, _ := os.Executable()
  os.Chdir(filepath.Dir(exePath))
  ```
  - [ ] Force au démarrage du service
  - [ ] Test: config.json trouvé dans ProgramData

- [ ] **Logs accessibles**
  - [ ] Service log dans `C:\ProgramData\Nimbus\logs\service.log`
  - [ ] GUI: bouton "Voir logs du service" (lecture seule)
  - [ ] Rotation: max 10 MB par fichier

### MSI - Finitions

- [ ] **Installation Silencieuse (Silent Install)**
  - [ ] **Approche: Config JSON pré-configuré** (propre pour AD/GPO)
    ```powershell
    # Déploiement avec config centralisée
    msiexec /i NimbusBackup.msi /qn CONFIGFILE="\\ad-server\deploy\nimbus\config.json"
    ```
  - [ ] Property WiX: `CONFIGFILE` (chemin vers config.json)
  - [ ] CustomAction WiX:
    - Si `CONFIGFILE` fourni → copier vers `C:\ProgramData\Nimbus\config.json`
    - Valider JSON avant copie (éviter corruption)
    - Log erreur si fichier inaccessible
  - [ ] Template config.json à fournir:
    ```json
    {
      "pbs_url": "https://pbs.example.com:8007",
      "auth_id": "backup-user@pbs",
      "secret": "your-api-token-secret",
      "datastore": "backup",
      "namespace": "clients",
      "backup_id": "",  // Vide = utilise hostname
      "backup_dirs": ["C:\\Users", "C:\\Important"],
      "exclusions": ["*.tmp", "*.log"],
      "schedule": {
        "enabled": true,
        "time": "02:00",
        "days": ["monday", "wednesday", "friday"]
      },
      "vss_enabled": true
    }
    ```
  - [ ] Test: install silencieux → service démarre avec config OK
  - [ ] Doc: guide déploiement GPO/Intune avec config.json

- [ ] **Code Signing**
  - [ ] Signer le binaire `.exe`
  - [ ] Signer le `.msi`
  - [ ] Certificat: à obtenir (DigiCert/Sectigo ~300€/an)

- [ ] **Désinstallation propre**
  - [ ] Script CustomAction: stop service avant uninstall
  - [ ] Nettoyer `C:\ProgramData\Nimbus` (option: garder config)

### Multi-jobs - Stabilisation
- [ ] **Queue management**
  - [ ] Pas de 2 jobs VSS simultanés
  - [ ] File d'attente FIFO
  - [ ] UI: afficher "En attente..." si queue pleine

- [ ] **Test de charge**
  - [ ] Lancer 5 jobs en même temps
  - [ ] Vérifier pas de corruption d'index PBS
  - [ ] RAM usage < 500 MB

---

## 🟢 P2 - NICE TO HAVE (Backlog)

### Windows - Compatibilité avancée
- [ ] **LongPath Support**
  - [ ] Ajouter manifeste: `<longPathAware>true</longPathAware>`
  - [ ] Test: backup d'un chemin >260 caractères

- [ ] **Gestion des locks**
  - [ ] Détecter fichier ouvert sans VSS
  - [ ] Erreur propre: "Fichier X verrouillé, activer VSS?"

### Chiffrement (Phase 2)
- [ ] **Key Management**
  - [ ] Génération clé asymétrique
  - [ ] Stockage: Windows Credential Manager (DPAPI)
  - [ ] Export: bouton "Sauvegarder clé de récupération"

- [ ] **GUI**
  - [ ] Checkbox "Activer chiffrement"
  - [ ] Warning: "Sans la clé, restauration impossible!"

### Restauration locale (Phase 3)
- [ ] **Navigateur de snapshots**
  - [ ] Liste des snapshots depuis PBS
  - [ ] Parcourir le catalog (lazy loading)
  - [ ] Sélection fichiers/dossiers

- [ ] **Download & Restore**
  - [ ] Bouton "Restaurer vers..."
  - [ ] Gestion conflits (Écraser/Renommer)
  - [ ] Extraction via service (droits admin)

### Mode Entreprise (Phase 4)
**Évolution naturelle grâce à l'architecture HTTP!**

- [ ] **HTTP Remote API** (extension du mode local)
  - [ ] Flag service: `--remote-api --bind 0.0.0.0 --port 18765`
  - [ ] Auth: Bearer token (généré, 32 chars aléatoires)
  - [ ] TLS: certificat auto-signé ou fourni par admin
  - [ ] Rate limiting: max 10 req/s par IP
  - [ ] Whitelist IPs: config pour restreindre accès

- [ ] **GUI Centrale**
  - [ ] Découverte machines: scan réseau ou import CSV
  - [ ] Dashboard: grille avec état de tous les backups
  - [ ] Actions groupées: "Backup tout le parc"
  - [ ] Alertes: notification si machine pas vue depuis 24h

---

## 🗑️ DROP (Ignoré pour l'instant)

- ❌ UUID machine (hostname suffit)
- ❌ Heartbeat vers API RDEM (overkill)
- ❌ go-msi (WiX fonctionne)
- ❌ Mount FUSE/WinFSP (restauration web suffit)

---

## 📅 Roadmap suggérée

**v0.2.x - Communication GUI-Service** (2-3 semaines)
- Named Pipes fonctionnel
- VSS via service uniquement
- GUI = télécommande

**v0.3.x - Robustesse Service** (2-3 semaines)
- VSS cleanup
- Logs propres
- MSI signed

**v0.4.x - Chiffrement** (3-4 semaines)
- Key management
- Encryption at rest

**v0.5.x - Restauration locale** (4-6 semaines)
- Browse snapshots
- Selective restore

**v1.0.0 - Production Ready** (3 mois total)
- Tout ce qui précède
- Doc complète
- Support officiel

---

**Dernière mise à jour:** 2026-03-19
**Mainteneur:** RDEM Systems
