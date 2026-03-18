# 📋 TODO Projet Nimbus Backup

## ✅ Phase 1 - Error Handling (COMPLÉTÉ)

- [x] PXAR callbacks retournent erreurs
- [x] HandleData() et Eof() avec gestion complète
- [x] Remplacement panic() par fatalError()
- [x] Wrapping erreurs avec context (fmt.Errorf)
- [x] Package logger avec tests
- [x] Package retry avec exponential backoff + tests
- [x] Package security avec validation/sanitization
- [x] Fix golangci-lint (errcheck, gosimple)
- [x] Renommer binaire → NimbusBackup dans CI

## ✅ Phase 2 - Tests (COMPLÉTÉ v0.1.2)

- [x] Tests logger (pkg/logger/logger_test.go)
- [x] Tests retry (pkg/retry/retry_test.go)
- [x] Tests chunking (pbscommon/chunking_test.go) - 15+ tests
- [ ] Tests PXAR (pbscommon/pxar_test.go) - TODO
- [x] Tests snapshot (snapshot/snapshot_test.go) - Windows VSS
- [ ] Target: 50%+ code coverage (en progression)

## ✅ Phase 3 - Security (COMPLÉTÉ v0.1.2)

- [x] Infrastructure security (pkg/security)
- [x] Intégrer validation dans tous les entry points
- [x] Ajouter sanitization à tous les logs
- [ ] Security audit complet avant v0.2 (review externe)
- [ ] TLS certificate pinning (optionnel, basse priorité)

## 🚀 Features v0.2 (ROADMAP Q2 2026)

### Priorité 1 (Critique)
- [ ] **Restore dans GUI**
  - Lire snapshots depuis PBS
  - Browser de fichiers PXAR
  - Extraction sélective
  - Progress bar restore

- [ ] **Gestion encryption client-side**
  - Wizard génération clé
  - Import/export clé
  - Stockage sécurisé keyfile
  - Warning backup clé utilisateur
  - Intégration PBS encryption

- [ ] **Code signing Authenticode**
  - Achat certificat
  - Intégration dans CI/CD
  - Signature automatique des releases
  - Re-enable machine backup après signing

### Priorité 2 (Important)
- [ ] **Traduction anglaise**
  - i18n infrastructure (react-i18next?)
  - Traduction interface GUI
  - Traduction messages d'erreur
  - Détection locale système
  - Sélecteur langue dans settings

- [ ] **Auto-update system**
  - Check GitHub releases API
  - Comparaison version
  - Download + verify nouvelle version
  - Update silencieux ou avec prompt
  - Rollback si échec

### Priorité 3 (Nice to have)
- [ ] **Scheduled backups**
  - UI planification (daily, weekly, cron)
  - Task scheduler Windows integration
  - Notifications succès/échec

- [ ] **Windows service mode**
  - Service installation
  - Background execution
  - Tray icon
  - Start with Windows

## 🔧 Améliorations techniques

- [ ] Replace Printf par structured logger partout
- [x] Retry logic sur toutes les opérations réseau (v0.1.1)
- [ ] Améliorer progress calculation (plus granulaire?)
- [ ] Compression settings configurables
- [ ] Chunk size configurable
- [ ] Bandwidth limiting (optionnel)

## 📝 Documentation

- [ ] README.md à jour avec features actuelles
- [ ] CONTRIBUTING.md pour contributeurs
- [ ] Architecture documentation
- [ ] API documentation (si library usage)

## 🌐 Site/Marketing

- [x] Page blog corrections éditoriales (EN ATTENTE PUBLICATION)
- [ ] Relecture page après publication
- [ ] Screenshots à jour v0.2
- [ ] Vidéo démo 30s-1min
- [ ] Guide utilisateur complet
- [ ] FAQ étendue

## 📦 CI/CD

- [x] GitLab pipeline fonctionnelle
- [ ] GitHub Actions setup (après push)
- [ ] Auto-release sur tag
- [ ] Build artifacts signing
- [ ] Multi-arch builds (arm64?)

## 🎯 Avant merge upstream (vers tizbac/proxmoxbackupclient_go)

- [ ] Phase 2 tests complète (50%+ coverage)
- [ ] Phase 3 security complète
- [ ] Documentation à jour
- [ ] CHANGELOG complet
- [ ] Code review interne
- [ ] Prepare PR avec description détaillée
- [ ] Discuter roadmap avec @tizbac

## 🏆 Objectifs v1.0 (Stable)

- [ ] Toutes features v0.2 complètes et testées
- [ ] 70%+ code coverage
- [ ] Zero critical bugs
- [ ] Code signing OK
- [ ] Auto-update OK
- [ ] Traduction EN/FR complète
- [ ] Documentation complète
- [ ] 50+ early adopters feedback intégré

---

## 🔥 NEXT IMMEDIATE ACTIONS

1. **Push sur GitHub** (maintenant)
2. **Relecture page blog** (après publication)
3. **Phase 2**: Continuer tests chunking/PXAR
4. **Feature restore GUI**: Commencer implémentation

---

**Dernière mise à jour**: 2026-03-18
**Version actuelle**: v0.1.0
**Prochaine release**: v0.2.0 (Q2 2026)
