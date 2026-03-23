# Nimbus Backup - TODO

## ✅ RÉCEMMENT COMPLÉTÉES (v0.1.78-v0.1.92)

### ~~Fix Bug Config Service~~ ✅ RÉSOLU (v0.1.81)
- [x] ~~HTTP API existe~~ ✓ (`gui/api/server.go`)
- [x] ~~Service Windows fonctionne~~ ✓ (`gui/service.go`)
- [x] ~~ReloadConfig avant backup~~ ✓ (v0.1.81)
- [x] ~~Scheduled backups~~ ✓ (v0.1.85+)
- [x] ~~System tray~~ ✓ (v0.1.80+)
- [x] ~~MSI installer~~ ✓ (v0.1.70+)
- [x] ~~Logs séparés GUI/Service~~ ✓ (v0.1.78+)

---

## ✅ RÉCEMMENT COMPLÉTÉES (2026-03-23)

### ~~VSS Cleanup au démarrage~~ ✅ RÉSOLU
- [x] ~~Fonction cleanupVSS() Windows~~ ✓ (`service/vss_cleanup_windows.go`)
- [x] ~~Appel au démarrage du service~~ ✓ (`service/main.go`)
- [x] ~~Log des snapshots supprimés~~ ✓
- [x] ~~Build tags Windows/autres plateformes~~ ✓

### ~~Multi-PBS Architecture~~ ✅ IMPLÉMENTÉ (Backend complet)
- [x] ~~Structure PBSServer avec validation~~ ✓ (`gui/pbs_server.go`)
- [x] ~~Config.PBSServers map~~ ✓ (`gui/config.go`)
- [x] ~~Migration auto legacy → multi-PBS~~ ✓
- [x] ~~CRUD methods (Add/Update/Delete/List)~~ ✓
- [x] ~~Job.PBSID référence serveur~~ ✓ (`gui/jobs.go`)
- [x] ~~Méthodes App exposées au frontend~~ ✓ (`gui/main.go`)
- [x] ~~Documentation complète~~ ✓ (`MULTI_PBS_GUIDE.md`)
- [ ] **Frontend GUI à développer** (liste serveurs, dropdown jobs)

### ~~MSI Uninstall Dialog~~ ✅ IMPLÉMENTÉ
- [x] ~~Propriété KEEP_CONFIG~~ ✓ (`installer/wix/Product.wxs`)
- [x] ~~Custom action DeleteConfigFolder~~ ✓
- [x] ~~Dialog personnalisé avec radio buttons~~ ✓
- [x] ~~Documentation tests~~ ✓ (`MSI_UNINSTALL_TEST.md`)
- [ ] **Tests sur Windows à faire**

---

## 🔴 P0 - CRITIQUE (À faire maintenant)

### ~~Multi-PBS Architecture~~ ✅ BACKEND COMPLET (2026-03-23)
**Use case:** Multi-datastore (C:\ → bigdata, C:\Users → ssd) + GUI distante

Backend implémenté :
- [x] ~~Structure config avec map de PBS~~ ✓
- [x] ~~Validation: au moins 1 PBS configuré~~ ✓
- [x] ~~Migration: config actuelle → pbs_servers["default"]~~ ✓
- [x] ~~Jobs référencent PBS par ID~~ ✓
- [x] ~~CRUD API exposée (Add/Update/Delete/List)~~ ✓
- [x] ~~Documentation complète~~ ✓ (MULTI_PBS_GUIDE.md)

**Frontend à développer** (2-3 jours) :
- [ ] Page "Serveurs PBS" (liste avec CRUD)
- [ ] Dropdown "Serveur PBS" dans formulaire backup
- [ ] Test connexion par PBS (bouton + indicateur 🟢/🔴)
- [ ] Migration jobs legacy vers PBSID

**Temps estimé:** 2-3 jours frontend

---

## 🟠 P1 - IMPORTANT (Architecture Entreprise)

---

## 🟠 P1 - IMPORTANT (Prochaines semaines)

### Service Windows - Robustesse
- [x] ~~**VSS Cleanup au démarrage**~~ ✅ FAIT (2026-03-23)
  - [x] ~~Appel dans `service.run()`~~ ✓
  - [x] ~~Log les shadows supprimées~~ ✓
  - [x] ~~Build tags Windows/Linux~~ ✓

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

- [x] ~~**Désinstallation avec choix config**~~ ✅ FAIT (2026-03-23)
  - [x] ~~Dialog WiX personnalisé~~ ✓
  - [x] ~~Propriété KEEP_CONFIG~~ ✓
  - [x] ~~CustomAction DeleteConfigFolder~~ ✓
  - [ ] **Tests sur Windows à faire**

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

### API Remote - Provisioning Distant (Phase 2)
**Use case:** MSP gère 100+ clients Nimbus depuis interface centrale

- [ ] **API Remote activable**
  ```json
  {
    "api": {
      "remote_enabled": false,  // Désactivé par défaut (sécurité)
      "bind_address": "0.0.0.0:18765",  // Si activé
      "auth_token": "generated-at-install",
      "tls_cert": "/path/to/cert.pem",  // Optionnel
      "allowed_ips": ["192.168.1.0/24"]  // Whitelist
    }
  }
  ```
  - [ ] Flag service: `--remote-api` pour activer
  - [ ] Auth: Bearer token (généré install, 32 chars)
  - [ ] TLS: Certificat auto-signé ou fourni
  - [ ] Rate limiting: max 10 req/s par IP
  - [ ] Whitelist IPs configurables

- [ ] **Endpoints Provisioning**
  - `GET /api/v1/info` - Info système (hostname, version, mode)
  - `GET /api/v1/pbs` - Liste serveurs PBS configurés
  - `POST /api/v1/pbs` - Ajouter serveur PBS
  - `PUT /api/v1/pbs/{id}` - Modifier serveur PBS
  - `DELETE /api/v1/pbs/{id}` - Supprimer serveur PBS
  - `POST /api/v1/pbs/{id}/test` - Test connexion
  - `GET /api/v1/jobs` - Liste jobs
  - `POST /api/v1/jobs` - Créer job
  - `PUT /api/v1/jobs/{id}` - Modifier job
  - `DELETE /api/v1/jobs/{id}` - Supprimer job
  - `POST /api/v1/backup` - Lancer backup manuel

- [ ] **GUI Centrale MSP** (Futur produit séparé)
  - Dashboard: grille avec tous les clients
  - Actions groupées: "Backup tout le parc"
  - Alertes: machine pas vue depuis 24h
  - Statistiques globales

**Temps estimé:** 2-3 semaines

### Multi-Serveurs PBS
**→ DÉPLACÉ EN P0** (voir "Multi-PBS Architecture" ci-dessus)

### Chiffrement (Phase 3)
- [ ] **Key Management**
  - [ ] Génération clé asymétrique
  - [ ] Stockage: Windows Credential Manager (DPAPI)
  - [ ] Export: bouton "Sauvegarder clé de récupération"

- [ ] **GUI**
  - [ ] Checkbox "Activer chiffrement"
  - [ ] Warning: "Sans la clé, restauration impossible!"

### ~~Restauration locale~~ ❌ HORS SCOPE
**Décision:** La restauration est déléguée au portail web [members.rdem-systems.com](https://members.rdem-systems.com)
- Les clients Nimbus utilisent l'interface PBS hébergée pour restaurer
- Pas de restauration dans le client desktop
- Cette section est retirée de la roadmap

### Mode Entreprise (Phase 5)
**→ DÉPLACÉ EN P1** (voir "API Remote - Provisioning Distant")

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
