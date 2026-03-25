const translations = {
  fr: {
    // Header
    appTitle: "Nimbus Backup",
    appSubtitle: "Client de sauvegarde pour Proxmox Backup Server - RDEM Systems",

    // Tabs
    tabServers: "Configuration PBS",
    tabBackup: "Sauvegarde",
    tabRestore: "Restauration",
    tabAbout: "À propos",

    // Common
    save: "Enregistrer",
    cancel: "Annuler",
    delete: "Supprimer",
    edit: "Modifier",
    add: "Ajouter",
    test: "Tester",
    start: "Démarrer",
    stop: "Arrêter",
    update: "Mettre à jour",
    actions: "Actions",
    name: "Nom",
    url: "URL",
    status: "Statut",
    online: "Online",
    offline: "Offline",
    testing: "Test...",
    untested: "Non testé",
    default: "Défaut",
    optional: "optionnel",

    // PBS Configuration
    serversTitle: "Configuration PBS",
    welcomeMessage: "Bienvenue !",
    welcomeText: "Configurez votre premier serveur PBS pour commencer les backups.",
    noPBSYet: "Vous n'avez pas encore de serveur PBS ?",
    orderStorage: "Commander du stockage Nimbus Backup",
    addYourServer: "Ajouter votre serveur PBS",
    addAnotherServer: "Ajouter un autre serveur PBS",
    editServer: "Modifier le serveur",
    addFirstServer: "Ajouter mon premier serveur",
    addServer: "Ajouter le serveur",
    configuredServers: "Serveurs configurés",
    setAsDefault: "Par défaut",

    // Server Form
    serverName: "Nom du serveur",
    serverID: "ID du serveur (auto-généré si vide)",
    serverIDPlaceholder: "pbs-ssd (laissez vide pour auto-génération)",
    serverURL: "URL du serveur PBS",
    authID: "Authentication ID",
    secret: "Secret (API Token)",
    datastore: "Datastore",
    namespace: "Namespace (optionnel)",
    certFingerprint: "Empreinte certificat SSL (optionnel)",
    description: "Description (optionnelle)",
    descriptionPlaceholder: "Stockage SSD pour backups critiques",

    // Multi-PBS
    multiPBSInfo: "Multi-PBS :",
    multiPBSText: "Gérez plusieurs serveurs PBS pour vos backups.",
    multiPBSExample: "Exemple : C:\\ → PBS big-data (lent, gros espace) | C:\\Users → PBS SSD (rapide, petit)",

    // Tips
    tipTitle: "Astuce :",
    tipAPIToken: "Obtenez votre API Token depuis l'interface PBS:",
    tipAPITokenPath: "Configuration → Access Control → API Tokens",

    // Backup
    backupTitle: "Sauvegarde",
    backupType: "Type de sauvegarde",
    backupTypeDirectory: "Répertoire (dossier spécifique)",
    backupTypeMachine: "Machine (disque complet)",
    executionMode: "Mode d'exécution",
    oneshotMode: "One-shot (maintenant)",
    oneshotModeShort: "Now",
    scheduledMode: "Planifié",
    scheduledModeShort: "Schedule",

    // Scheduling
    schedulingConfig: "Configuration de la planification",
    editMode: "Mode édition",
    editModeText: "Modifiez les paramètres et cliquez sur \"Mettre à jour\"",
    dailyExecutionTime: "Heure d'exécution quotidienne",
    runAtStartup: "Exécuter aussi au démarrage de la machine",
    schedulingInfo: "Le backup sera exécuté automatiquement chaque jour à",
    andAtStartup: "Et également à chaque démarrage du système.",

    // Backup Form
    directoriesToBackup: "Répertoires à sauvegarder (un par ligne)",
    physicalDisksToBackup: "Disques physiques à sauvegarder",
    loadingDisks: "Chargement des disques disponibles...",
    filesToExclude: "Fichiers à exclure (un par ligne, optionnel)",
    backupID: "Backup ID",
    backupIDPlaceholder: "Laissez vide pour utiliser le hostname",
    useVSS: "Utiliser VSS (Windows Shadow Copy)",
    vssAdminRequired: "VSS nécessite des privilèges administrateur.",
    vssAdminHint: "Redémarrez l'application en tant qu'administrateur (clic droit → Exécuter en tant qu'administrateur) pour utiliser VSS.",
    vssServiceAvailable: "VSS disponible via le service.",
    vssServiceHint: "Le service Windows tourne avec les privilèges nécessaires pour VSS.",

    // Backup Progress
    backupProgress: "Progression du backup",
    timeRemaining: "Temps restant:",
    speed: "Vitesse:",
    elapsedTime: "Temps écoulé:",
    backupInProgress: "Sauvegarde en cours...",
    startBackup: "Démarrer la sauvegarde",
    saveSchedule: "Enregistrer la planification",
    updateSchedule: "Mettre à jour la planification",
    stopBackup: "Arrêter",
    cancelEdit: "Annuler",

    // Scheduled Jobs
    scheduledJobs: "Jobs planifiés",
    editJob: "Éditer",
    deleteJob: "Supprimer",
    editModeInfo: "Mode édition - modifiez et sauvegardez",

    // Backup History
    backupHistory: "Historique des sauvegardes (dernières 6)",
    rerun: "Relancer",
    configLoaded: "Configuration chargée, lancez le backup",

    // Restore
    restoreTitle: "Restauration",
    backupIDToRestore: "Backup ID à restaurer",
    listSnapshots: "Lister les snapshots disponibles",
    availableSnapshots: "Snapshots disponibles",
    noSnapshotFound: "Aucun snapshot trouvé",
    restore: "Restaurer",
    restoreInfo: "Restauration :",
    restoreInfoText: "Sélectionnez d'abord un Backup ID, puis listez les snapshots disponibles.",
    restoreInfoText2: "Vous pourrez ensuite choisir un snapshot spécifique et le répertoire de destination pour la restauration.",
    restoreDestPrompt: "Chemin de destination pour la restauration:",

    // About
    aboutTitle: "À propos",
    version: "Version",
    orderStorageCTA: "Commander du stockage Nimbus Backup",
    features: "Fonctionnalités",
    featuresList: {
      directories: "Sauvegarde répertoires & disques",
      machine: "Machine complète (C:\\, D:\\, etc.)",
      restore: "Restauration snapshots",
      vss: "Support VSS (Shadow Copy)",
      dedup: "Déduplication & compression",
      modern: "Interface Wails moderne"
    },
    technology: "Technologie",
    techList: {
      wails: "Wails v2 (Go + React)",
      performance: "Performance native",
      interface: "Interface moderne",
      logs: "Logs de debug intégrés",
      nogpu: "Pas de dépendance GPU"
    },
    copyright: "© 2026 RDEM Systems",
    basedOn: "Basé sur proxmoxbackupclient_go par tizbac",
    techStack: "Interface Wails + React + Vite",

    // Status Messages
    statusDiskError: "Erreur lors de la détection des disques:",
    statusServerLoadError: "Erreur chargement serveurs:",
    statusServerAdded: "Serveur PBS ajouté",
    statusServerUpdated: "Serveur PBS mis à jour",
    statusServerDeleted: "Serveur PBS supprimé",
    statusServerSetDefault: "Serveur \"{id}\" défini par défaut",
    statusConnectionSuccess: "Connexion au serveur \"{id}\" réussie",
    statusConnectionFailed: "Connexion échouée:",
    statusConfigSaved: "Configuration enregistrée",
    statusConnectionOK: "Connexion réussie !",
    statusConfigLoaded: "Configuration chargée depuis le fichier",
    statusInvalidJSON: "Erreur : fichier JSON invalide",
    statusBackupComplete: "Tous les backups partiels terminés avec succès ({done}/{total})",
    statusSplitError: "Erreur split backup:",
    statusSchedulingUnavailable: "Fonction de planification non disponible",
    statusBackupStarting: "Démarrage de la sauvegarde...",
    statusBackupRunning: "Sauvegarde en cours...",
    statusRestoring: "Restauration du snapshot {time}...",
    statusRestoreComplete: "Restauration terminée !",
    statusJobDeleted: "Job supprimé",
    statusEditCancelled: "Édition annulée",
    statusError: "Erreur:",
    statusConfirm: "Êtes-vous sûr ?",
    confirmDeleteServer: "Voulez-vous vraiment supprimer le serveur PBS \"{id}\" ?",
  },
  en: {
    // Header
    appTitle: "Nimbus Backup",
    appSubtitle: "Backup client for Proxmox Backup Server - RDEM Systems",

    // Tabs
    tabServers: "PBS Configuration",
    tabBackup: "Backup",
    tabRestore: "Restore",
    tabAbout: "About",

    // Common
    save: "Save",
    cancel: "Cancel",
    delete: "Delete",
    edit: "Edit",
    add: "Add",
    test: "Test",
    start: "Start",
    stop: "Stop",
    update: "Update",
    actions: "Actions",
    name: "Name",
    url: "URL",
    status: "Status",
    online: "Online",
    offline: "Offline",
    testing: "Testing...",
    untested: "Not tested",
    default: "Default",
    optional: "optional",

    // PBS Configuration
    serversTitle: "PBS Configuration",
    welcomeMessage: "Welcome!",
    welcomeText: "Configure your first PBS server to start backups.",
    noPBSYet: "Don't have a PBS server yet?",
    orderStorage: "Order Nimbus Backup storage",
    addYourServer: "Add your PBS server",
    addAnotherServer: "Add another PBS server",
    editServer: "Edit server",
    addFirstServer: "Add my first server",
    addServer: "Add server",
    configuredServers: "Configured servers",
    setAsDefault: "Set as default",

    // Server Form
    serverName: "Server name",
    serverID: "Server ID (auto-generated if empty)",
    serverIDPlaceholder: "pbs-ssd (leave empty for auto-generation)",
    serverURL: "PBS server URL",
    authID: "Authentication ID",
    secret: "Secret (API Token)",
    datastore: "Datastore",
    namespace: "Namespace (optional)",
    certFingerprint: "SSL certificate fingerprint (optional)",
    description: "Description (optional)",
    descriptionPlaceholder: "SSD storage for critical backups",

    // Multi-PBS
    multiPBSInfo: "Multi-PBS:",
    multiPBSText: "Manage multiple PBS servers for your backups.",
    multiPBSExample: "Example: C:\\ → PBS big-data (slow, large space) | C:\\Users → PBS SSD (fast, small)",

    // Tips
    tipTitle: "Tip:",
    tipAPIToken: "Get your API Token from the PBS interface:",
    tipAPITokenPath: "Configuration → Access Control → API Tokens",

    // Backup
    backupTitle: "Backup",
    backupType: "Backup type",
    backupTypeDirectory: "Directory (specific folder)",
    backupTypeMachine: "Machine (full disk)",
    executionMode: "Execution mode",
    oneshotMode: "One-shot (now)",
    oneshotModeShort: "Now",
    scheduledMode: "Scheduled",
    scheduledModeShort: "Schedule",

    // Scheduling
    schedulingConfig: "Scheduling configuration",
    editMode: "Edit mode",
    editModeText: "Modify settings and click \"Update\"",
    dailyExecutionTime: "Daily execution time",
    runAtStartup: "Also run at machine startup",
    schedulingInfo: "Backup will run automatically every day at",
    andAtStartup: "And also at every system startup.",

    // Backup Form
    directoriesToBackup: "Directories to backup (one per line)",
    physicalDisksToBackup: "Physical disks to backup",
    loadingDisks: "Loading available disks...",
    filesToExclude: "Files to exclude (one per line, optional)",
    backupID: "Backup ID",
    backupIDPlaceholder: "Leave empty to use hostname",
    useVSS: "Use VSS (Windows Shadow Copy)",
    vssAdminRequired: "VSS requires administrator privileges.",
    vssAdminHint: "Restart the application as administrator (right-click → Run as administrator) to use VSS.",
    vssServiceAvailable: "VSS available via service.",
    vssServiceHint: "The Windows service runs with the necessary privileges for VSS.",

    // Backup Progress
    backupProgress: "Backup progress",
    timeRemaining: "Time remaining:",
    speed: "Speed:",
    elapsedTime: "Elapsed time:",
    backupInProgress: "Backup in progress...",
    startBackup: "Start backup",
    saveSchedule: "Save schedule",
    updateSchedule: "Update schedule",
    stopBackup: "Stop",
    cancelEdit: "Cancel",

    // Scheduled Jobs
    scheduledJobs: "Scheduled jobs",
    editJob: "Edit",
    deleteJob: "Delete",
    editModeInfo: "Edit mode - modify and save",

    // Backup History
    backupHistory: "Backup history (last 6)",
    rerun: "Rerun",
    configLoaded: "Configuration loaded, start backup",

    // Restore
    restoreTitle: "Restore",
    backupIDToRestore: "Backup ID to restore",
    listSnapshots: "List available snapshots",
    availableSnapshots: "Available snapshots",
    noSnapshotFound: "No snapshot found",
    restore: "Restore",
    restoreInfo: "Restore:",
    restoreInfoText: "First select a Backup ID, then list available snapshots.",
    restoreInfoText2: "You can then choose a specific snapshot and the destination directory for restoration.",
    restoreDestPrompt: "Destination path for restoration:",

    // About
    aboutTitle: "About",
    version: "Version",
    orderStorageCTA: "Order Nimbus Backup storage",
    features: "Features",
    featuresList: {
      directories: "Directory & disk backup",
      machine: "Full machine (C:\\, D:\\, etc.)",
      restore: "Snapshot restoration",
      vss: "VSS support (Shadow Copy)",
      dedup: "Deduplication & compression",
      modern: "Modern Wails interface"
    },
    technology: "Technology",
    techList: {
      wails: "Wails v2 (Go + React)",
      performance: "Native performance",
      interface: "Modern interface",
      logs: "Integrated debug logs",
      nogpu: "No GPU dependency"
    },
    copyright: "© 2026 RDEM Systems",
    basedOn: "Based on proxmoxbackupclient_go by tizbac",
    techStack: "Wails + React + Vite interface",

    // Status Messages
    statusDiskError: "Error detecting disks:",
    statusServerLoadError: "Error loading servers:",
    statusServerAdded: "PBS server added",
    statusServerUpdated: "PBS server updated",
    statusServerDeleted: "PBS server deleted",
    statusServerSetDefault: "Server \"{id}\" set as default",
    statusConnectionSuccess: "Connection to server \"{id}\" successful",
    statusConnectionFailed: "Connection failed:",
    statusConfigSaved: "Configuration saved",
    statusConnectionOK: "Connection successful!",
    statusConfigLoaded: "Configuration loaded from file",
    statusInvalidJSON: "Error: invalid JSON file",
    statusBackupComplete: "All partial backups completed successfully ({done}/{total})",
    statusSplitError: "Split backup error:",
    statusSchedulingUnavailable: "Scheduling function unavailable",
    statusBackupStarting: "Starting backup...",
    statusBackupRunning: "Backup in progress...",
    statusRestoring: "Restoring snapshot {time}...",
    statusRestoreComplete: "Restoration complete!",
    statusJobDeleted: "Job deleted",
    statusEditCancelled: "Edit cancelled",
    statusError: "Error:",
    statusConfirm: "Are you sure?",
    confirmDeleteServer: "Do you really want to delete the PBS server \"{id}\"?",
  }
}

export default translations
