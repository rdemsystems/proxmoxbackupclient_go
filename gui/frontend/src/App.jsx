import { useState, useEffect } from 'react'
import { useTranslation } from './i18n/i18nContext'
import LanguageSwitcher from './components/LanguageSwitcher'

// Wails runtime imports (will be available when built with Wails)
let GetConfigWithHostname, SaveConfig, TestConnection, StartBackup, ListSnapshots, RestoreSnapshot, ListPhysicalDisks, GetVersion, EventsOn
let SaveScheduledJob, UpdateScheduledJob, GetScheduledJobs, DeleteScheduledJob, GetJobHistory, GetSystemInfo, GetLastBackupDirs
// Multi-PBS functions
let ListPBSServers, GetPBSServer, AddPBSServer, UpdatePBSServer, DeletePBSServer, SetDefaultPBSServer, GetDefaultPBSID, TestPBSConnection

// Check if we're running in Wails
if (window.go) {
  GetConfigWithHostname = window.go.main.App.GetConfigWithHostname
  SaveConfig = window.go.main.App.SaveConfig
  TestConnection = window.go.main.App.TestConnection
  StartBackup = window.go.main.App.StartBackup
  ListSnapshots = window.go.main.App.ListSnapshots
  RestoreSnapshot = window.go.main.App.RestoreSnapshot
  ListPhysicalDisks = window.go.main.App.ListPhysicalDisks
  GetVersion = window.go.main.App.GetVersion
  SaveScheduledJob = window.go.main.App.SaveScheduledJob
  UpdateScheduledJob = window.go.main.App.UpdateScheduledJob
  GetScheduledJobs = window.go.main.App.GetScheduledJobs
  DeleteScheduledJob = window.go.main.App.DeleteScheduledJob
  GetJobHistory = window.go.main.App.GetJobHistory
  GetSystemInfo = window.go.main.App.GetSystemInfo
  GetLastBackupDirs = window.go.main.App.GetLastBackupDirs
  // Multi-PBS
  ListPBSServers = window.go.main.App.ListPBSServers
  GetPBSServer = window.go.main.App.GetPBSServer
  AddPBSServer = window.go.main.App.AddPBSServer
  UpdatePBSServer = window.go.main.App.UpdatePBSServer
  DeletePBSServer = window.go.main.App.DeletePBSServer
  SetDefaultPBSServer = window.go.main.App.SetDefaultPBSServer
  GetDefaultPBSID = window.go.main.App.GetDefaultPBSID
  TestPBSConnection = window.go.main.App.TestPBSConnection
}

// Wails events
if (window.runtime) {
  EventsOn = window.runtime.EventsOn
}

function App() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('servers')
  const [hostname, setHostname] = useState('')
  const [appVersion, setAppVersion] = useState('dev')
  const [systemInfo, setSystemInfo] = useState({ mode: 'Standalone', is_admin: false, service_available: false })
  const [config, setConfig] = useState({
    baseurl: '',
    certfingerprint: '',
    authid: '',
    secret: '',
    datastore: '',
    namespace: '',
    backupdir: '',
    'backup-id': '',
    usevss: true
  })

  // Multi-PBS states
  const [pbsServers, setPbsServers] = useState([])
  const [defaultPBSID, setDefaultPBSID] = useState('')
  const [selectedPBSID, setSelectedPBSID] = useState('')
  const [editingServer, setEditingServer] = useState(null)
  const [serverFormData, setServerFormData] = useState({
    id: '',
    name: '',
    baseurl: '',
    certfingerprint: '',
    authid: '',
    secret: '',
    datastore: '',
    namespace: '',
    description: ''
  })
  const [serverStatus, setServerStatus] = useState({}) // Map of server ID -> connection status

  const [backupType, setBackupType] = useState('directory')
  const [backupDirs, setBackupDirs] = useState('')
  const [selectedDrives, setSelectedDrives] = useState([])
  const [physicalDisks, setPhysicalDisks] = useState([])
  const [excludeList, setExcludeList] = useState('')
  const [progress, setProgress] = useState(0)

  // Scheduling states
  const [backupMode, setBackupMode] = useState('oneshot') // 'oneshot' or 'scheduled'
  const [scheduleTime, setScheduleTime] = useState('02:00')
  const [runAtStartup, setRunAtStartup] = useState(false)
  const [scheduledJobs, setScheduledJobs] = useState([])
  const [jobHistory, setJobHistory] = useState([])
  const [editingJobId, setEditingJobId] = useState(null) // Track which job is being edited
  const [backupStats, setBackupStats] = useState({
    startTime: null,
    lastUpdate: null,
    lastPercent: 0,
    speed: 0,
    eta: null
  })
  const [status, setStatus] = useState({ message: '', type: '', visible: false })

  const [snapshots, setSnapshots] = useState([])
  const [restoreBackupId, setRestoreBackupId] = useState('')
  const [showSnapshots, setShowSnapshots] = useState(false)

  // Update restoreBackupId when config or hostname changes
  useEffect(() => {
    if (!restoreBackupId && (config['backup-id'] || hostname)) {
      setRestoreBackupId(config['backup-id'] || hostname)
    }
  }, [config['backup-id'], hostname])

  // Load physical disks when switching to machine mode (DISABLED FOR NOW)
  /*
  useEffect(() => {
    if (backupType === 'machine' && ListPhysicalDisks && physicalDisks.length === 0) {
      ListPhysicalDisks().then(disks => {
        setPhysicalDisks(disks)
        // Select first disk by default
        if (disks.length > 0 && selectedDrives.length === 0) {
          setSelectedDrives([disks[0].path])
        }
      }).catch(err => {
        showStatus(`❌ Erreur lors de la détection des disques: ${err}`, 'error')
      })
    }
  }, [backupType])
  */

  // Listen to backup events
  useEffect(() => {
    if (!EventsOn) return

    const unsubProgress = EventsOn('backup:progress', (data) => {
      const now = Date.now()
      const percent = Math.round(data.percent)
      setProgress(percent)
      showStatus(`🔄 ${data.message}`, 'info')

      // Calculate speed and ETA
      setBackupStats(prev => {
        const startTime = prev.startTime || now
        const lastUpdate = prev.lastUpdate || now
        const timeDiff = (now - lastUpdate) / 1000 // seconds
        const percentDiff = percent - prev.lastPercent

        // Calculate speed (percent per second)
        let speed = prev.speed
        if (timeDiff > 0 && percentDiff > 0) {
          speed = percentDiff / timeDiff
        }

        // Calculate ETA (seconds remaining)
        let eta = null
        if (speed > 0 && percent < 100) {
          const remainingPercent = 100 - percent
          eta = Math.round(remainingPercent / speed)
        }

        return {
          startTime,
          lastUpdate: now,
          lastPercent: percent,
          speed,
          eta
        }
      })
    })

    const unsubComplete = EventsOn('backup:complete', (data) => {
      setProgress(data.success ? 100 : 0)
      setBackupStats({ startTime: null, lastUpdate: null, lastPercent: 0, speed: 0, eta: null })
      showStatus(data.success ? '✅ ' + data.message : '❌ ' + data.message, data.success ? 'success' : 'error')

      // Add to job history
      const historyEntry = {
        id: Date.now().toString(),
        name: `Backup ${config['backup-id'] || hostname}`,
        timestamp: new Date().toISOString(),
        status: data.success ? 'success' : 'failed',
        message: data.message,
        backupDirs: backupDirs.split('\n').map(d => d.trim()).filter(d => d),
        backupId: config['backup-id'] || hostname,
        useVSS: config.usevss
      }
      setJobHistory(prev => [historyEntry, ...prev].slice(0, 20)) // Keep last 20 entries
    })

    return () => {
      if (unsubProgress) unsubProgress()
      if (unsubComplete) unsubComplete()
    }
  }, [])

  // Load config with hostname on mount
  useEffect(() => {
    const loadData = async () => {
      try {
        // Load version
        if (GetVersion) {
          const version = await GetVersion()
          setAppVersion(version || 'dev')
        }

        // Load system info (mode, admin status, service availability)
        if (GetSystemInfo) {
          const sysInfo = await GetSystemInfo()
          setSystemInfo(sysInfo || { mode: 'Standalone', is_admin: false, service_available: false })
        }

        // Load last backup directories to pre-fill the form
        if (GetLastBackupDirs) {
          const lastDirs = await GetLastBackupDirs()
          if (lastDirs && lastDirs.length > 0) {
            setBackupDirs(lastDirs.join('\n'))
          }
        }

        if (GetConfigWithHostname) {
          const data = await GetConfigWithHostname()
          if (data) {
            // Extract hostname
            const hn = data.hostname || ''
            setHostname(hn)

            // Set config (hostname is already in backup-id if needed)
            setConfig({
              baseurl: data.baseurl || '',
              certfingerprint: data.certfingerprint || '',
              authid: data.authid || '',
              secret: data.secret || '',
              datastore: data.datastore || '',
              namespace: data.namespace || '',
              backupdir: data.backupdir || '',
              'backup-id': data['backup-id'] || hn,
              usevss: data.usevss !== undefined ? data.usevss : true
            })

            // Initialize backupDirs from config if available
            if (data.backupdir) {
              setBackupDirs(data.backupdir)
            }
          }
        }
      } catch (err) {
        console.error('Failed to load config:', err)
      }
    }

    loadData()
  }, [])

  // Load scheduled jobs and history on mount
  useEffect(() => {
    const loadSchedulerData = async () => {
      try {
        if (GetScheduledJobs) {
          const jobs = await GetScheduledJobs()
          setScheduledJobs(jobs || [])
        }

        if (GetJobHistory) {
          const history = await GetJobHistory()
          setJobHistory(history || [])
        }
      } catch (err) {
        console.error('Failed to load scheduler data:', err)
      }
    }

    loadSchedulerData()

    // Refresh history every 10 seconds to update status of running jobs
    const intervalId = setInterval(() => {
      if (GetJobHistory) {
        GetJobHistory().then(history => {
          setJobHistory(history || [])
        }).catch(err => {
          console.error('Failed to refresh job history:', err)
        })
      }
    }, 10000) // 10 seconds

    return () => clearInterval(intervalId)
  }, [])

  // Load PBS servers on mount
  useEffect(() => {
    const loadPBSServers = async () => {
      try {
        if (ListPBSServers) {
          const servers = await ListPBSServers()
          setPbsServers(servers || [])
        }

        if (GetDefaultPBSID) {
          const defaultID = await GetDefaultPBSID()
          setDefaultPBSID(defaultID || '')
          setSelectedPBSID(defaultID || '')
        }
      } catch (err) {
        console.error('Failed to load PBS servers:', err)
      }
    }

    loadPBSServers()
  }, [])

  const showStatus = (message, type) => {
    setStatus({ message, type, visible: true })
    setTimeout(() => {
      setStatus(s => ({ ...s, visible: false }))
    }, 5000)
  }

  // ==================== MULTI-PBS HANDLERS ====================

  const loadPBSServers = async () => {
    try {
      if (ListPBSServers) {
        const servers = await ListPBSServers()
        setPbsServers(servers || [])
      }
      if (GetDefaultPBSID) {
        const defaultID = await GetDefaultPBSID()
        setDefaultPBSID(defaultID || '')
      }
    } catch (err) {
      console.error('Failed to load PBS servers:', err)
      showStatus(`❌ ${t('statusServerLoadError')} ${err}`, 'error')
    }
  }

  const handleAddPBSServer = async () => {
    if (!AddPBSServer) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    try {
      // Generate ID from name if not provided
      if (!serverFormData.id) {
        serverFormData.id = serverFormData.name.toLowerCase().replace(/[^a-z0-9]/g, '-')
      }

      await AddPBSServer(serverFormData)
      showStatus(`✅ ${t('statusServerAdded')}`, 'success')

      // Reset form and reload
      setServerFormData({
        id: '',
        name: '',
        baseurl: '',
        certfingerprint: '',
        authid: '',
        secret: '',
        datastore: '',
        namespace: '',
        description: ''
      })
      setEditingServer(null)
      await loadPBSServers()
    } catch (err) {
      showStatus(`❌ Erreur: ${err}`, 'error')
    }
  }

  const handleUpdatePBSServer = async () => {
    if (!UpdatePBSServer) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    try {
      await UpdatePBSServer(serverFormData)
      showStatus(`✅ ${t('statusServerUpdated')}`, 'success')

      // Reset form and reload
      setServerFormData({
        id: '',
        name: '',
        baseurl: '',
        certfingerprint: '',
        authid: '',
        secret: '',
        datastore: '',
        namespace: '',
        description: ''
      })
      setEditingServer(null)
      await loadPBSServers()
    } catch (err) {
      showStatus(`❌ Erreur: ${err}`, 'error')
    }
  }

  const handleDeletePBSServer = async (id) => {
    if (!DeletePBSServer) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    if (!confirm(t('confirmDeleteServer').replace('{id}', id))) {
      return
    }

    try {
      await DeletePBSServer(id)
      showStatus(`✅ ${t('statusServerDeleted')}`, 'success')
      await loadPBSServers()
    } catch (err) {
      showStatus(`❌ Erreur: ${err}`, 'error')
    }
  }

  const handleSetDefaultPBS = async (id) => {
    if (!SetDefaultPBSServer) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    try {
      await SetDefaultPBSServer(id)
      setDefaultPBSID(id)
      showStatus(`✅ ${t('statusServerSetDefault').replace('{id}', id)}`, 'success')
    } catch (err) {
      showStatus(`❌ Erreur: ${err}`, 'error')
    }
  }

  const handleTestPBSConnection = async (id) => {
    if (!TestPBSConnection) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    try {
      setServerStatus(prev => ({ ...prev, [id]: 'testing' }))
      await TestPBSConnection(id)
      setServerStatus(prev => ({ ...prev, [id]: 'online' }))
      showStatus(`✅ ${t('statusConnectionSuccess').replace('{id}', id)}`, 'success')
    } catch (err) {
      setServerStatus(prev => ({ ...prev, [id]: 'offline' }))
      showStatus(`❌ ${t('statusConnectionFailed')} ${err}`, 'error')
    }
  }

  const handleEditServer = (server) => {
    setServerFormData(server)
    setEditingServer(server.id)
  }

  const handleCancelEdit = () => {
    setServerFormData({
      id: '',
      name: '',
      baseurl: '',
      certfingerprint: '',
      authid: '',
      secret: '',
      datastore: '',
      namespace: '',
      description: ''
    })
    setEditingServer(null)
  }

  // ==================== END MULTI-PBS HANDLERS ====================

  const handleSaveConfig = async () => {
    if (!SaveConfig) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    try {
      // Trim all string values to remove whitespace (with safe fallback for undefined)
      const trimmedConfig = {
        baseurl: (config.baseurl || '').trim(),
        certfingerprint: (config.certfingerprint || '').trim(),
        authid: (config.authid || '').trim(),
        secret: (config.secret || '').trim(),
        datastore: (config.datastore || '').trim(),
        namespace: (config.namespace || '').trim(),
        backupdir: (config.backupdir || '').trim(),
        'backup-id': (config['backup-id'] || '').trim() || hostname, // Use hostname if empty
        usevss: config.usevss !== undefined ? config.usevss : true
      }
      await SaveConfig(trimmedConfig)
      setConfig(trimmedConfig)
      showStatus(`✅ ${t('statusConfigSaved')}`, 'success')
    } catch (err) {
      showStatus(`❌ Erreur : ${err}`, 'error')
    }
  }

  const handleTestConnection = async () => {
    if (!TestConnection) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    try {
      // Test with current form values (no need to save first)
      const testConfig = {
        baseurl: (config.baseurl || '').trim(),
        certfingerprint: (config.certfingerprint || '').trim(),
        authid: (config.authid || '').trim(),
        secret: (config.secret || '').trim(),
        datastore: (config.datastore || '').trim(),
        namespace: (config.namespace || '').trim(),
        backupdir: (config.backupdir || '').trim(),
        'backup-id': (config['backup-id'] || '').trim() || hostname, // Use hostname if empty
        usevss: config.usevss !== undefined ? config.usevss : true
      }
      await TestConnection(testConfig)
      showStatus(`✅ ${t('statusConnectionOK')}`, 'success')
    } catch (err) {
      showStatus(`❌ ${err}`, 'error')
    }
  }

  const handleLoadConfigFile = (e) => {
    const file = e.target.files[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (evt) => {
      try {
        const loadedConfig = JSON.parse(evt.target.result)
        setConfig(loadedConfig)
        showStatus(`✅ ${t('statusConfigLoaded')}`, 'success')
      } catch (err) {
        showStatus(`❌ ${t('statusInvalidJSON')}`, 'error')
      }
    }
    reader.readAsText(file)
  }

  // Execute split backup for large volumes
  const executeSplitBackup = async (dirList, analysis) => {
    if (!window.go || !window.go.main.App.CreateBackupSplitPlan) {
      showStatus('❌ Split backup not available', 'error')
      return
    }

    try {
      showStatus('📋 Création du plan de découpage...', 'info')
      const splitPlan = await window.go.main.App.CreateBackupSplitPlan(
        dirList,
        config['backup-id'] || hostname
      )

      showStatus(`🔄 Lancement de ${splitPlan.length} backups partiels...`, 'info')

      // Execute split jobs sequentially
      for (let i = 0; i < splitPlan.length; i++) {
        const job = splitPlan[i]
        showStatus(
          `📦 Backup ${job.index}/${job.total_jobs}: ${job.size_fmt}...`,
          'info'
        )

        try {
          await StartBackup(
            backupType,
            job.folders,
            selectedDrives,
            excludeList.split('\n').filter(l => l.trim()),
            job.backup_id,
            config.usevss
          )

          // Wait for completion (simplified - in production, use event polling)
          showStatus(
            `✅ Backup ${job.index}/${job.total_jobs} terminé`,
            'success'
          )
        } catch (err) {
          showStatus(
            `❌ Backup ${job.index}/${job.total_jobs} échoué: ${err}`,
            'error'
          )

          const retry = window.confirm(
            `Le backup ${job.index}/${job.total_jobs} a échoué.\n\n` +
            `Voulez-vous réessayer ce backup avant de continuer?`
          )

          if (retry) {
            i-- // Retry same job
          } else {
            throw new Error(`Split backup ${job.index} failed`)
          }
        }
      }

      showStatus(
        `🎉 Tous les backups partiels terminés avec succès (${splitPlan.length}/${splitPlan.length})`,
        'success'
      )
    } catch (err) {
      showStatus(`❌ Erreur split backup: ${err}`, 'error')
    }
  }

  const handleStartBackup = async () => {
    if (!StartBackup) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    // Parse backup directories (one per line)
    const dirList = backupDirs.split('\n').map(d => d.trim()).filter(d => d)

    if (backupType === 'directory' && dirList.length === 0) {
      showStatus('❌ Au moins un répertoire requis', 'error')
      return
    }

    if (backupType === 'machine' && selectedDrives.length === 0) {
      showStatus('❌ Au moins un disque requis', 'error')
      return
    }

    // Analyze backup size for auto-split (only for directory backups in oneshot mode)
    if (backupType === 'directory' && backupMode === 'oneshot' && window.go && window.go.main.App.AnalyzeBackup) {
      try {
        showStatus('📊 Analyse de la taille du backup...', 'info')
        const analysis = await window.go.main.App.AnalyzeBackup(dirList)

        if (analysis.should_split) {
          const confirmSplit = window.confirm(
            `📦 Backup volumineux détecté (${analysis.total_size_fmt})\n\n` +
            `Pour améliorer la fiabilité et la vitesse, voulez-vous le découper en ` +
            `${analysis.suggested_jobs} backups plus petits (~100 GB chacun) ?\n\n` +
            `✅ Avantages:\n` +
            `  • Résistance aux pannes (retry ciblé)\n` +
            `  • Progression visible\n` +
            `  • Plus rapide en cas d'échec\n\n` +
            `Les backups seront consolidés automatiquement une fois terminés.`
          )

          if (confirmSplit) {
            // User accepted split - execute split backup
            await executeSplitBackup(dirList, analysis)
            return
          }
          // User declined - continue with normal backup below
        }
      } catch (err) {
        // Analysis failed - continue with normal backup
        console.warn('Backup analysis failed:', err)
      }
    }

    // Scheduled mode - save or update job instead of executing immediately
    if (backupMode === 'scheduled') {
      if (!SaveScheduledJob || !UpdateScheduledJob) {
        showStatus('❌ Fonction de planification non disponible', 'error')
        return
      }

      const jobData = {
        id: editingJobId || Date.now().toString(),
        name: `Backup ${config['backup-id'] || hostname}`,
        scheduleTime: scheduleTime,
        runAtStartup: runAtStartup,
        backupDirs: dirList,
        backupId: config['backup-id'],
        useVSS: config.usevss,
        backupType: backupType,
        excludeList: excludeList.split('\n').filter(l => l.trim())
      }

      // Save or update to backend
      try {
        if (editingJobId) {
          // Update existing job
          await UpdateScheduledJob(jobData)
          setScheduledJobs(scheduledJobs.map(j => j.id === editingJobId ? jobData : j))
          showStatus(`✅ Backup modifié pour ${scheduleTime}`, 'success')
          setEditingJobId(null)
        } else {
          // Create new job
          await SaveScheduledJob(jobData)
          setScheduledJobs([...scheduledJobs, jobData])
          showStatus(`✅ Backup planifié pour ${scheduleTime}`, 'success')
        }
        // Reset form after save
        setScheduleTime('02:00')
        setRunAtStartup(false)
        setBackupDirs('')
      } catch (err) {
        showStatus(`❌ Erreur: ${err}`, 'error')
      }
      return
    }

    // One-shot mode - execute immediately
    showStatus(`🚀 ${t('statusBackupStarting')}`, 'info')
    setProgress(5)

    try {
      await StartBackup(
        backupType,
        dirList,
        selectedDrives,
        excludeList.split('\n').filter(l => l.trim()),
        config['backup-id'],
        config.usevss
      )
      // Backup started in background - progress will be shown via events
      showStatus(`⏳ ${t('statusBackupRunning')}`, 'info')
    } catch (err) {
      setProgress(0)
      showStatus(`❌ ${err}`, 'error')
    }
  }

  const handleListSnapshots = async () => {
    if (!ListSnapshots) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    if (!restoreBackupId) {
      showStatus('❌ Backup ID requis', 'error')
      return
    }

    showStatus('🔍 Recherche des snapshots...', 'info')

    try {
      const snaps = await ListSnapshots(restoreBackupId)
      setSnapshots(snaps || [])
      setShowSnapshots(true)
      showStatus(`✅ ${snaps.length} snapshot(s) trouvé(s)`, 'success')
    } catch (err) {
      showStatus(`❌ ${err}`, 'error')
    }
  }

  const handleRestoreSnapshot = async (snapshotId, time) => {
    if (!RestoreSnapshot) {
      showStatus('❌ Wails runtime non disponible', 'error')
      return
    }

    const destPath = prompt(t('restoreDestPrompt'), 'C:\\Restore')
    if (!destPath) return

    showStatus(`🔄 ${t('statusRestoring').replace('{time}', time)}`, 'info')

    try {
      await RestoreSnapshot(snapshotId, destPath)
      showStatus(`✅ ${t('statusRestoreComplete')}`, 'success')
    } catch (err) {
      showStatus(`❌ ${err}`, 'error')
    }
  }

  return (
    <>
      <div className="header">
        <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
          <div>
            <h1>🛡️ {t('appTitle')}</h1>
            <p>{t('appSubtitle')}</p>
          </div>
          <LanguageSwitcher />
        </div>
      </div>

      <div className="container">
        <div className="tabs">
          <div className={`tab ${activeTab === 'servers' ? 'active' : ''}`} onClick={() => setActiveTab('servers')}>
            {t('tabServers')}
          </div>
          <div className={`tab ${activeTab === 'backup' ? 'active' : ''}`} onClick={() => setActiveTab('backup')}>
            {t('tabBackup')}
          </div>
          <div className={`tab ${activeTab === 'restore' ? 'active' : ''}`} onClick={() => setActiveTab('restore')}>
            {t('tabRestore')}
          </div>
          <div className={`tab ${activeTab === 'about' ? 'active' : ''}`} onClick={() => setActiveTab('about')}>
            {t('tabAbout')}
          </div>
        </div>

        {/* PBS Configuration Tab */}
        <div className={`tab-content ${activeTab === 'servers' ? 'active' : ''}`}>
          <h2>🖥️ {t('serversTitle')}</h2>

          {/* Show form first if no servers configured */}
          {pbsServers.length === 0 ? (
            <>
              <div className="info-box" style={{marginBottom: '20px', backgroundColor: '#eef2ff', borderLeft: '4px solid #667eea'}}>
                👋 <strong>{t('welcomeMessage')}</strong> {t('welcomeText')}<br/>
                {!config.baseurl && (
                  <>
                    <br/>
                    <strong>📦 {t('noPBSYet')}</strong><br/>
                    <a
                      href={`https://nimbus.rdem-systems.com/choisir-mon-backup/?utm_source=NimbusGui&utm_medium=tooling&utm_campaign=version-${appVersion}&utm_content=first-setup`}
                      target="_blank"
                      rel="noopener noreferrer"
                      style={{color: '#667eea', fontWeight: 'bold', textDecoration: 'underline'}}
                    >
                      {t('orderStorage')} →
                    </a>
                  </>
                )}
              </div>

              {/* Add Server Form - Prominent when no servers */}
              <div className="card">
                <h3>➕ {t('addYourServer')}</h3>
              <table style={{width: '100%', marginTop: '15px'}}>
                <thead>
                  <tr>
                    <th>{t('name')}</th>
                    <th>{t('url')}</th>
                    <th>{t('datastore')}</th>
                    <th>{t('status')}</th>
                    <th>{t('actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {pbsServers.map(server => (
                    <tr key={server.id}>
                      <td>
                        <strong>{server.name}</strong>
                        {server.id === defaultPBSID && <span style={{marginLeft: '5px', color: '#fbbf24'}}>⭐ {t('default')}</span>}
                        {server.description && <div style={{fontSize: '0.85em', color: '#999'}}>{server.description}</div>}
                      </td>
                      <td>{server.baseurl}</td>
                      <td>{server.datastore}/{server.namespace || '-'}</td>
                      <td>
                        {serverStatus[server.id] === 'testing' && <span style={{color: '#3b82f6'}}>🔄 {t('testing')}</span>}
                        {serverStatus[server.id] === 'online' && <span style={{color: '#10b981'}}>🟢 {t('online')}</span>}
                        {serverStatus[server.id] === 'offline' && <span style={{color: '#ef4444'}}>🔴 {t('offline')}</span>}
                        {!serverStatus[server.id] && <span style={{color: '#999'}}>⚪ {t('untested')}</span>}
                      </td>
                      <td>
                        <button onClick={() => handleTestPBSConnection(server.id)} style={{marginRight: '5px', padding: '5px 10px', fontSize: '0.9em'}}>
                          🔍 {t('test')}
                        </button>
                        <button onClick={() => handleEditServer(server)} style={{marginRight: '5px', padding: '5px 10px', fontSize: '0.9em'}}>
                          ✏️ {t('edit')}
                        </button>
                        {server.id !== defaultPBSID && (
                          <button onClick={() => handleSetDefaultPBS(server.id)} style={{marginRight: '5px', padding: '5px 10px', fontSize: '0.9em', backgroundColor: '#fbbf24'}}>
                            ⭐ {t('setAsDefault')}
                          </button>
                        )}
                        <button onClick={() => handleDeletePBSServer(server.id)} style={{padding: '5px 10px', fontSize: '0.9em', backgroundColor: '#ef4444', color: 'white'}}>
                          🗑️ {t('delete')}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          {/* Add/Edit Server Form */}
          <div className="card">
            <h3>{editingServer ? `✏️ ${t('editServer')}` : `➕ ${t('addYourServer')}`}</h3>

            <div className="form-group">
              <label>{t('serverName')}</label>
              <input
                type="text"
                value={serverFormData.name}
                onChange={(e) => setServerFormData({...serverFormData, name: e.target.value})}
                placeholder="SSD Rapide"
              />
            </div>

            {!editingServer && (
              <div className="form-group">
                <label>{t('serverID')}</label>
                <input
                  type="text"
                  value={serverFormData.id}
                  onChange={(e) => setServerFormData({...serverFormData, id: e.target.value})}
                  placeholder="pbs-ssd (laissez vide pour auto-génération)"
                />
              </div>
            )}

            <div className="form-group">
              <label>{t('serverURL')}</label>
              <input
                type="text"
                value={serverFormData.baseurl}
                onChange={(e) => setServerFormData({...serverFormData, baseurl: e.target.value})}
                placeholder="https://pbs-ssd.example.com:8007"
              />
            </div>

            <div className="form-group">
              <label>{t('authID')}</label>
              <input
                type="text"
                value={serverFormData.authid}
                onChange={(e) => setServerFormData({...serverFormData, authid: e.target.value})}
                placeholder="backup@pbs!token-name"
              />
            </div>

            <div className="form-group">
              <label>{t('secret')}</label>
              <input
                type="password"
                value={serverFormData.secret}
                onChange={(e) => setServerFormData({...serverFormData, secret: e.target.value})}
                placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
              />
            </div>

            <div className="form-group">
              <label>{t('datastore')}</label>
              <input
                type="text"
                value={serverFormData.datastore}
                onChange={(e) => setServerFormData({...serverFormData, datastore: e.target.value})}
                placeholder="ssd-fast"
              />
            </div>

            <div className="form-group">
              <label>{t('namespace')}</label>
              <input
                type="text"
                value={serverFormData.namespace}
                onChange={(e) => setServerFormData({...serverFormData, namespace: e.target.value})}
                placeholder="clients"
              />
            </div>

            <div className="form-group">
              <label>{t('certFingerprint')}</label>
              <input
                type="text"
                value={serverFormData.certfingerprint}
                onChange={(e) => setServerFormData({...serverFormData, certfingerprint: e.target.value})}
                placeholder="AA:BB:CC:DD:..."
              />
            </div>

            <div className="form-group">
              <label>{t('description')}</label>
              <textarea
                value={serverFormData.description}
                onChange={(e) => setServerFormData({...serverFormData, description: e.target.value})}
                placeholder="Stockage SSD pour backups critiques"
                rows="2"
              />
            </div>

            <div style={{display: 'flex', gap: '10px', marginTop: '20px'}}>
              {editingServer ? (
                <>
                  <button onClick={handleUpdatePBSServer} style={{flex: 1}}>
                    💾 {t('update')}
                  </button>
                  <button onClick={handleCancelEdit} style={{flex: 1, backgroundColor: '#999'}}>
                    ❌ {t('cancel')}
                  </button>
                </>
              ) : (
                <button onClick={handleAddPBSServer} style={{flex: 1}}>
                  ➕ {t('addFirstServer')}
                </button>
              )}
            </div>

            <div className="info-box" style={{marginTop: '20px'}}>
              💡 <strong>{t('tipTitle')}</strong> {t('tipAPIToken')}<br/>
              {t('tipAPITokenPath')}
            </div>
          </div>
            </>
          ) : (
            <>
              {/* Multi-PBS info for users with existing servers */}
              <div className="info-box" style={{marginBottom: '20px'}}>
                💡 <strong>{t('multiPBSInfo')}</strong> {t('multiPBSText')}<br/>
                {t('multiPBSExample')}
              </div>

              {/* Server List */}
              <div className="card" style={{marginBottom: '20px'}}>
                <h3>{t('configuredServers')} ({pbsServers.length})</h3>

                <table style={{width: '100%', marginTop: '15px'}}>
                  <thead>
                    <tr>
                      <th>Nom</th>
                      <th>URL</th>
                      <th>Datastore</th>
                      <th>Statut</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {pbsServers.map(server => (
                      <tr key={server.id}>
                        <td>
                          <strong>{server.name}</strong>
                          {server.id === defaultPBSID && <span style={{marginLeft: '5px', color: '#fbbf24'}}>⭐ {t('default')}</span>}
                          {server.description && <div style={{fontSize: '0.85em', color: '#999'}}>{server.description}</div>}
                        </td>
                        <td>{server.baseurl}</td>
                        <td>{server.datastore}/{server.namespace || '-'}</td>
                        <td>
                          {serverStatus[server.id] === 'testing' && <span style={{color: '#3b82f6'}}>🔄 Test...</span>}
                          {serverStatus[server.id] === 'online' && <span style={{color: '#10b981'}}>🟢 Online</span>}
                          {serverStatus[server.id] === 'offline' && <span style={{color: '#ef4444'}}>🔴 Offline</span>}
                          {!serverStatus[server.id] && <span style={{color: '#999'}}>⚪ Non testé</span>}
                        </td>
                        <td>
                          <button onClick={() => handleTestPBSConnection(server.id)} style={{marginRight: '5px', padding: '5px 10px', fontSize: '0.9em'}}>
                            🔍 Tester
                          </button>
                          <button onClick={() => handleEditServer(server)} style={{marginRight: '5px', padding: '5px 10px', fontSize: '0.9em'}}>
                            ✏️ Modifier
                          </button>
                          {server.id !== defaultPBSID && (
                            <button onClick={() => handleSetDefaultPBS(server.id)} style={{marginRight: '5px', padding: '5px 10px', fontSize: '0.9em', backgroundColor: '#fbbf24'}}>
                              ⭐ Par défaut
                            </button>
                          )}
                          <button onClick={() => handleDeletePBSServer(server.id)} style={{padding: '5px 10px', fontSize: '0.9em', backgroundColor: '#ef4444', color: 'white'}}>
                            🗑️ Supprimer
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Add/Edit Server Form */}
              <div className="card">
                <h3>{editingServer ? `✏️ ${t('editServer')}` : `➕ ${t('addAnotherServer')}`}</h3>

                <div className="form-group">
                  <label>{t('serverName')}</label>
                  <input
                    type="text"
                    value={serverFormData.name}
                    onChange={(e) => setServerFormData({...serverFormData, name: e.target.value})}
                    placeholder="SSD Rapide"
                  />
                </div>

                {!editingServer && (
                  <div className="form-group">
                    <label>{t('serverID')}</label>
                    <input
                      type="text"
                      value={serverFormData.id}
                      onChange={(e) => setServerFormData({...serverFormData, id: e.target.value})}
                      placeholder="pbs-ssd (laissez vide pour auto-génération)"
                    />
                  </div>
                )}

                <div className="form-group">
                  <label>{t('serverURL')}</label>
                  <input
                    type="text"
                    value={serverFormData.baseurl}
                    onChange={(e) => setServerFormData({...serverFormData, baseurl: e.target.value})}
                    placeholder="https://pbs-ssd.example.com:8007"
                  />
                </div>

                <div className="form-group">
                  <label>{t('authID')}</label>
                  <input
                    type="text"
                    value={serverFormData.authid}
                    onChange={(e) => setServerFormData({...serverFormData, authid: e.target.value})}
                    placeholder="backup@pbs!token-name"
                  />
                </div>

                <div className="form-group">
                  <label>{t('secret')}</label>
                  <input
                    type="password"
                    value={serverFormData.secret}
                    onChange={(e) => setServerFormData({...serverFormData, secret: e.target.value})}
                    placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                  />
                </div>

                <div className="form-group">
                  <label>{t('datastore')}</label>
                  <input
                    type="text"
                    value={serverFormData.datastore}
                    onChange={(e) => setServerFormData({...serverFormData, datastore: e.target.value})}
                    placeholder="ssd-fast"
                  />
                </div>

                <div className="form-group">
                  <label>{t('namespace')}</label>
                  <input
                    type="text"
                    value={serverFormData.namespace}
                    onChange={(e) => setServerFormData({...serverFormData, namespace: e.target.value})}
                    placeholder="clients"
                  />
                </div>

                <div className="form-group">
                  <label>{t('certFingerprint')}</label>
                  <input
                    type="text"
                    value={serverFormData.certfingerprint}
                    onChange={(e) => setServerFormData({...serverFormData, certfingerprint: e.target.value})}
                    placeholder="AA:BB:CC:DD:..."
                  />
                </div>

                <div className="form-group">
                  <label>{t('description')}</label>
                  <textarea
                    value={serverFormData.description}
                    onChange={(e) => setServerFormData({...serverFormData, description: e.target.value})}
                    placeholder="Stockage SSD pour backups critiques"
                    rows="2"
                  />
                </div>

                <div style={{display: 'flex', gap: '10px', marginTop: '20px'}}>
                  {editingServer ? (
                    <>
                      <button onClick={handleUpdatePBSServer} style={{flex: 1}}>
                        💾 Mettre à jour
                      </button>
                      <button onClick={handleCancelEdit} style={{flex: 1, backgroundColor: '#999'}}>
                        ❌ Annuler
                      </button>
                    </>
                  ) : (
                    <button onClick={handleAddPBSServer} style={{flex: 1}}>
                      ➕ {t('addServer')}
                    </button>
                  )}
                </div>
              </div>
            </>
          )}

          {status.visible && activeTab === 'servers' && (
            <div className={`status ${status.type} visible`}>{status.message}</div>
          )}
        </div>

        {/* Backup Tab */}
        <div className={`tab-content ${activeTab === 'backup' ? 'active' : ''}`}>
          <h2>{t('backupTitle')}</h2>

          <div className="form-group">
            <label>{t('backupType')}</label>
            <select value={backupType} onChange={(e) => setBackupType(e.target.value)}>
              <option value="directory">📁 {t('backupTypeDirectory')}</option>
              {/* <option value="machine">💾 {t('backupTypeMachine')}</option> */}
            </select>
          </div>

          {/* Backup Mode Toggle */}
          <div className="form-group">
            <label>{t('executionMode')}</label>
            <div style={{display: 'flex', gap: '10px', marginTop: '10px'}}>
              <button
                onClick={() => setBackupMode('oneshot')}
                style={{
                  flex: 1,
                  padding: '10px',
                  backgroundColor: backupMode === 'oneshot' ? '#667eea' : '#e2e8f0',
                  color: backupMode === 'oneshot' ? 'white' : '#4a5568',
                  border: 'none',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  fontWeight: 'bold'
                }}
              >
                <span className="compact-text-long">⚡ {t('oneshotMode')}</span>
                <span className="compact-text-short">⚡ {t('oneshotModeShort')}</span>
              </button>
              <button
                onClick={() => setBackupMode('scheduled')}
                style={{
                  flex: 1,
                  padding: '10px',
                  backgroundColor: backupMode === 'scheduled' ? '#667eea' : '#e2e8f0',
                  color: backupMode === 'scheduled' ? 'white' : '#4a5568',
                  border: 'none',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  fontWeight: 'bold'
                }}
              >
                <span className="compact-text-long">📅 {t('scheduledMode')}</span>
                <span className="compact-text-short">📅 {t('scheduledModeShort')}</span>
              </button>
            </div>
          </div>

          {/* Scheduling Options */}
          {backupMode === 'scheduled' && (
            <div className="card" style={{marginTop: '20px', padding: '20px'}}>
              <h3 style={{marginTop: 0}}>⏰ {t('schedulingConfig')}</h3>

              {editingJobId && (
                <div className="info-box" style={{backgroundColor: '#fff3cd', borderColor: '#ffc107', marginBottom: '15px'}}>
                  ✏️ <strong>{t('editMode')}</strong> - {t('editModeText')}
                </div>
              )}

              <div className="form-group">
                <label>{t('dailyExecutionTime')}</label>
                <input
                  type="time"
                  value={scheduleTime}
                  onChange={(e) => setScheduleTime(e.target.value)}
                  style={{width: '200px', padding: '10px', fontSize: '16px'}}
                />
              </div>

              <div className="form-group">
                <label style={{display: 'flex', alignItems: 'center', gap: '10px', cursor: 'pointer'}}>
                  <input
                    type="checkbox"
                    checked={runAtStartup}
                    onChange={(e) => setRunAtStartup(e.target.checked)}
                    style={{width: '20px', height: '20px', cursor: 'pointer'}}
                  />
                  <span>🚀 {t('runAtStartup')}</span>
                </label>
              </div>

              <div className="info-box" style={{backgroundColor: '#eef2ff'}}>
                💡 {t('schedulingInfo')} <strong>{scheduleTime}</strong>
                {runAtStartup && <><br/>{t('andAtStartup')}</>}
              </div>
            </div>
          )}

          {backupType === 'directory' ? (
            <div className="form-group">
              <label>{t('directoriesToBackup')}</label>
              <textarea
                value={backupDirs}
                onChange={(e) => {
                  setBackupDirs(e.target.value)
                  // Update config.backupdir with first directory for compatibility
                  const dirs = e.target.value.split('\n').map(d => d.trim()).filter(d => d)
                  setConfig({...config, backupdir: dirs[0] || ''})
                }}
                rows="4"
                placeholder="C:\Data&#10;C:\Users&#10;D:\Documents"
              />
            </div>
          ) : (
            <>
              <div className="form-group">
                <label>{t('physicalDisksToBackup')}</label>
                {physicalDisks.length === 0 ? (
                  <div style={{padding: '10px', backgroundColor: '#f8f9fa', borderRadius: '4px'}}>
                    🔍 {t('loadingDisks')}
                  </div>
                ) : (
                  <div style={{display: 'flex', flexDirection: 'column', gap: '8px'}}>
                    {physicalDisks.map(disk => (
                      <label key={disk.path} style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
                        <input
                          type="checkbox"
                          checked={selectedDrives.includes(disk.path)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedDrives([...selectedDrives, disk.path])
                            } else {
                              setSelectedDrives(selectedDrives.filter(d => d !== disk.path))
                            }
                          }}
                        />
                        {disk.label}
                      </label>
                    ))}
                  </div>
                )}
              </div>

              <div className="form-group">
                <label>{t('filesToExclude')}</label>
                <textarea
                  value={excludeList}
                  onChange={(e) => setExcludeList(e.target.value)}
                  rows="4"
                  placeholder="*.tmp&#10;*.log&#10;C:\Windows\Temp"
                />
              </div>
            </>
          )}

          <div className="form-group">
            <label>{t('backupID')}</label>
            <input
              type="text"
              value={config['backup-id']}
              onChange={(e) => setConfig({...config, 'backup-id': e.target.value})}
              placeholder={t('backupIDPlaceholder')}
            />
          </div>

          <div className="form-group">
            <label>
              <input
                type="checkbox"
                checked={config.usevss}
                onChange={(e) => setConfig({...config, usevss: e.target.checked})}
              />
              {t('useVSS')}
            </label>
            {config.usevss && systemInfo.mode === 'Standalone' && !systemInfo.is_admin && (
              <div className="info-box" style={{marginTop: '10px', backgroundColor: '#fff3cd', borderColor: '#ffc107'}}>
                ⚠️ <strong>{t('vssAdminRequired')}</strong><br/>
                {t('vssAdminHint')}
              </div>
            )}
            {config.usevss && systemInfo.service_available && (
              <div className="info-box" style={{marginTop: '10px', backgroundColor: '#d1ecf1', borderColor: '#bee5eb'}}>
                ℹ️ <strong>{t('vssServiceAvailable')}</strong><br/>
                {t('vssServiceHint')}
              </div>
            )}
          </div>

          {progress > 0 && progress < 100 && (
            <div style={{marginTop: '20px', marginBottom: '20px', padding: '15px', backgroundColor: '#f8f9fa', borderRadius: '8px', border: '1px solid #dee2e6'}}>
              <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: '10px'}}>
                <strong style={{fontSize: '15px'}}>📊 {t('backupProgress')}</strong>
                <span style={{fontSize: '18px', fontWeight: 'bold', color: '#0066cc'}}>{progress}%</span>
              </div>

              <div className="progress" style={{height: '30px', marginBottom: '12px'}}>
                <div
                  className="progress-bar"
                  style={{
                    width: `${progress}%`,
                    fontSize: '14px',
                    lineHeight: '30px',
                    transition: 'width 0.3s ease',
                    fontWeight: 'bold'
                  }}
                >
                  {progress}%
                </div>
              </div>

              <div style={{display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '10px'}}>
                {backupStats.eta !== null && (
                  <div style={{fontSize: '13px', color: '#495057'}}>
                    ⏱️ <strong>{t('timeRemaining')}</strong> {Math.floor(backupStats.eta / 60)}m {backupStats.eta % 60}s
                  </div>
                )}
                {backupStats.speed > 0 && (
                  <div style={{fontSize: '13px', color: '#495057'}}>
                    ⚡ <strong>{t('speed')}</strong> {backupStats.speed.toFixed(1)}%/s
                  </div>
                )}
                {backupStats.startTime && (
                  <div style={{fontSize: '13px', color: '#495057'}}>
                    ⏰ <strong>{t('elapsedTime')}</strong> {Math.floor((Date.now() - backupStats.startTime) / 1000)}s
                  </div>
                )}
              </div>

              {status.message && status.type === 'info' && (
                <div style={{marginTop: '10px', padding: '8px', backgroundColor: '#fff', borderRadius: '4px', fontSize: '13px', color: '#666', border: '1px solid #e9ecef'}}>
                  {status.message}
                </div>
              )}
            </div>
          )}

          <button className="btn" onClick={handleStartBackup} disabled={progress > 0 && progress < 100}>
            {backupMode === 'oneshot'
              ? (progress > 0 && progress < 100 ? `⏳ ${t('backupInProgress')}` : `🚀 ${t('startBackup')}`)
              : (editingJobId ? `✏️ ${t('updateSchedule')}` : `💾 ${t('saveSchedule')}`)
            }
          </button>
          {backupMode === 'oneshot' && (
            <button className="btn btn-secondary" onClick={() => setProgress(0)} disabled={progress === 0}>{t('stopBackup')}</button>
          )}
          {backupMode === 'scheduled' && editingJobId && (
            <button className="btn btn-secondary" onClick={() => {
              setEditingJobId(null)
              setScheduleTime('02:00')
              setRunAtStartup(false)
              setBackupDirs('')
              setExcludeList('')
              setBackupType('directory')
              setActiveTab('scheduled')
              showStatus(`✖️ ${t('statusEditCancelled')}`, 'info')
            }}>
              ✖️ {t('cancel')}
            </button>
          )}

          {/* Scheduled Jobs List */}
          {backupMode === 'scheduled' && scheduledJobs.length > 0 && (
            <div className="card" style={{marginTop: '30px'}}>
              <h3 style={{marginTop: 0}}>📅 {t('scheduledJobs')}</h3>
              {scheduledJobs.map(job => (
                <div key={job.id} style={{
                  padding: '15px',
                  marginBottom: '10px',
                  backgroundColor: '#f8f9fa',
                  borderRadius: '8px',
                  border: '1px solid #dee2e6'
                }}>
                  <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
                    <div>
                      <strong>{job.name}</strong>
                      <div style={{fontSize: '14px', color: '#6c757d', marginTop: '5px'}}>
                        ⏰ {job.scheduleTime} {job.runAtStartup && '• 🚀 Au démarrage'}
                      </div>
                      <div style={{fontSize: '13px', color: '#6c757d', marginTop: '3px'}}>
                        📁 {job.backupDirs.join(', ')}
                      </div>
                    </div>
                    <div style={{display: 'flex', gap: '10px'}}>
                      <button
                        className="btn"
                        style={{padding: '8px 15px', fontSize: '14px'}}
                        onClick={() => {
                          // Load job data into form for editing
                          setEditingJobId(job.id)
                          setBackupMode('scheduled')
                          setScheduleTime(job.scheduleTime)
                          setRunAtStartup(job.runAtStartup)
                          setBackupDirs(job.backupDirs.join('\n'))
                          setConfig({...config, 'backup-id': job.backupId, usevss: job.useVSS})
                          setBackupType(job.backupType)
                          setExcludeList(job.excludeList.join('\n'))
                          // Switch to backup tab to show the form
                          setActiveTab('backup')
                          showStatus(`✏️ ${t('editModeInfo')}`, 'info')
                          window.scrollTo({top: 0, behavior: 'smooth'})
                        }}
                      >
                        ✏️ {t('editJob')}
                      </button>
                      <button
                        className="btn btn-secondary"
                        style={{padding: '8px 15px', fontSize: '14px'}}
                        onClick={async () => {
                          try {
                            await DeleteScheduledJob(job.id)
                            setScheduledJobs(scheduledJobs.filter(j => j.id !== job.id))
                            showStatus(t('statusJobDeleted'), 'success')
                            // Cancel edit mode if deleting the job being edited
                            if (editingJobId === job.id) {
                              setEditingJobId(null)
                            }
                          } catch (err) {
                            showStatus(`❌ Erreur: ${err}`, 'error')
                          }
                        }}
                      >
                        🗑️ {t('deleteJob')}
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Job History */}
          {jobHistory.length > 0 && (
            <div className="card" style={{marginTop: '30px'}}>
              <h3 style={{marginTop: 0}}>📜 {t('backupHistory')}</h3>
              <div style={{maxHeight: '400px', overflowY: 'auto'}}>
                {jobHistory.slice(0, 6).map(job => (
                  <div key={job.id} style={{
                    padding: '15px',
                    marginBottom: '10px',
                    backgroundColor: job.status === 'success' ? '#d4edda' : job.status === 'failed' ? '#f8d7da' : '#fff3cd',
                    borderRadius: '8px',
                    border: `1px solid ${job.status === 'success' ? '#c3e6cb' : job.status === 'failed' ? '#f5c6cb' : '#ffeaa7'}`
                  }}>
                    <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
                      <div style={{flex: 1}}>
                        <div style={{display: 'flex', alignItems: 'center', gap: '10px'}}>
                          <span style={{fontSize: '20px'}}>
                            {job.status === 'success' ? '✅' : job.status === 'failed' ? '❌' : '⏳'}
                          </span>
                          <strong>{job.name}</strong>
                        </div>
                        <div style={{fontSize: '13px', color: '#6c757d', marginTop: '5px', marginLeft: '30px'}}>
                          🕐 {new Date(job.timestamp).toLocaleString('fr-FR')}
                        </div>
                        {job.message && (
                          <div style={{fontSize: '13px', color: '#495057', marginTop: '5px', marginLeft: '30px'}}>
                            💬 {job.message}
                          </div>
                        )}
                      </div>
                      {job.status === 'failed' && (
                        <button
                          className="btn"
                          style={{padding: '8px 15px', fontSize: '14px'}}
                          onClick={() => {
                            // Re-run failed job
                            setBackupDirs(job.backupDirs.join('\n'))
                            setConfig({...config, 'backup-id': job.backupId, usevss: job.useVSS})
                            showStatus(t('configLoaded'), 'success')
                            window.scrollTo({top: 0, behavior: 'smooth'})
                          }}
                        >
                          🔄 {t('rerun')}
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {status.visible && activeTab === 'backup' && (
            <div className={`status ${status.type} visible`}>{status.message}</div>
          )}
        </div>

        {/* Restore Tab */}
        <div className={`tab-content ${activeTab === 'restore' ? 'active' : ''}`}>
          <h2>{t('restoreTitle')}</h2>

          {/* BETA Warning */}
          <div style={{
            backgroundColor: '#FEF3C7',
            border: '2px solid #F59E0B',
            borderRadius: '8px',
            padding: '12px',
            marginBottom: '20px',
            color: '#92400E'
          }}>
            <strong>⚠️ BETA FEATURE</strong>
            <p style={{margin: '8px 0 0 0', fontSize: '14px'}}>
              La restauration est en phase BETA. Supporte actuellement :
              <br/>✅ Fichiers et dossiers simples
              <br/>✅ Permissions basiques
              <br/>❌ Symlinks, ACLs, attributs étendus (prochainement)
            </p>
          </div>

          <div className="form-group">
            <label>{t('backupIDToRestore')}</label>
            <input
              type="text"
              value={restoreBackupId || hostname}
              onChange={(e) => setRestoreBackupId(e.target.value)}
              placeholder={hostname || "hostname ou ID personnalisé"}
            />
          </div>

          <button className="btn" onClick={handleListSnapshots}>📋 {t('listSnapshots')}</button>

          {showSnapshots && (
            <div style={{marginTop: '20px'}}>
              <h3>{t('availableSnapshots')}</h3>
              <div className="grid">
                {snapshots.length === 0 ? (
                  <p style={{color: '#718096'}}>{t('noSnapshotFound')}</p>
                ) : (
                  snapshots.map((snap, idx) => (
                    <div key={idx} className="card" style={{cursor: 'pointer'}}>
                      <h3>📸 {snap.time}</h3>
                      <p style={{color: '#718096', fontSize: '14px', marginTop: '5px'}}>
                        ID: {snap.id}<br/>
                        Type: {snap.type || 'N/A'}
                      </p>
                      <button
                        className="btn"
                        style={{marginTop: '10px', width: '100%'}}
                        onClick={() => handleRestoreSnapshot(snap.id, snap.time)}
                      >
                        {t('restore')}
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}

          <div className="info-box" style={{marginTop: '20px'}}>
            💡 <strong>{t('restoreInfo')}</strong> {t('restoreInfoText')}<br/>
            {t('restoreInfoText2')}
          </div>

          {status.visible && activeTab === 'restore' && (
            <div className={`status ${status.type} visible`}>{status.message}</div>
          )}
        </div>

        {/* About Tab */}
        <div className={`tab-content ${activeTab === 'about' ? 'active' : ''}`}>
          <h2 style={{textAlign: 'center'}}>{t('aboutTitle')}</h2>

          <img
            src="https://nimbus.rdem-systems.com/logo.webp"
            alt="Nimbus Backup"
            className="logo"
            onError={(e) => e.target.style.display = 'none'}
          />

          <div style={{textAlign: 'center', marginTop: '30px'}}>
            <h3>Nimbus Backup</h3>
            <p style={{color: '#718096', margin: '10px 0'}}>{t('version')} {appVersion}</p>

            {/* Upsell CTA */}
            <div style={{margin: '20px 0'}}>
              <a
                href={`https://nimbus.rdem-systems.com/choisir-mon-backup/?utm_source=NimbusGui&utm_medium=tooling&utm_campaign=version-${appVersion}&utm_content=version-${appVersion}`}
                target="_blank"
                rel="noopener noreferrer"
                style={{
                  display: 'inline-block',
                  padding: '12px 24px',
                  backgroundColor: '#667eea',
                  color: 'white',
                  textDecoration: 'none',
                  borderRadius: '8px',
                  fontWeight: 'bold',
                  transition: 'background-color 0.3s'
                }}
                onMouseEnter={(e) => e.target.style.backgroundColor = '#5568d3'}
                onMouseLeave={(e) => e.target.style.backgroundColor = '#667eea'}
              >
                📦 {t('orderStorageCTA')}
              </a>
            </div>

            <div className="grid" style={{marginTop: '30px', textAlign: 'left'}}>
              <div className="card">
                <h3>✅ {t('features')}</h3>
                <ul style={{lineHeight: 2, marginLeft: '20px'}}>
                  <li>{t('featuresList.directories')}</li>
                  <li>{t('featuresList.machine')}</li>
                  <li>{t('featuresList.restore')}</li>
                  <li>{t('featuresList.vss')}</li>
                  <li>{t('featuresList.dedup')}</li>
                  <li>{t('featuresList.modern')}</li>
                </ul>
              </div>

              <div className="card">
                <h3>🚀 {t('technology')}</h3>
                <ul style={{lineHeight: 2, marginLeft: '20px'}}>
                  <li>{t('techList.wails')}</li>
                  <li>{t('techList.performance')}</li>
                  <li>{t('techList.interface')}</li>
                  <li>{t('techList.logs')}</li>
                  <li>{t('techList.nogpu')}</li>
                </ul>
              </div>
            </div>

            <p style={{marginTop: '30px'}}>
              <strong>{t('copyright')}</strong><br/>
              <a href="https://nimbus.rdem-systems.com" style={{color: '#667eea'}}>nimbus.rdem-systems.com</a>
            </p>

            <p style={{marginTop: '20px', color: '#718096', fontSize: '12px'}}>
              {t('basedOn')}<br/>
              {t('techStack')}
            </p>
          </div>
        </div>
      </div>
    </>
  )
}

export default App
