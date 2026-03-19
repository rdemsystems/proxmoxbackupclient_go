import { useState, useEffect } from 'react'

// Wails runtime imports (will be available when built with Wails)
let GetConfigWithHostname, SaveConfig, TestConnection, StartBackup, ListSnapshots, RestoreSnapshot, ListPhysicalDisks, GetVersion, EventsOn
let SaveScheduledJob, UpdateScheduledJob, GetScheduledJobs, DeleteScheduledJob, GetJobHistory

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
}

// Wails events
if (window.runtime) {
  EventsOn = window.runtime.EventsOn
}

function App() {
  const [activeTab, setActiveTab] = useState('config')
  const [hostname, setHostname] = useState('')
  const [appVersion, setAppVersion] = useState('dev')
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

  const showStatus = (message, type) => {
    setStatus({ message, type, visible: true })
    setTimeout(() => {
      setStatus(s => ({ ...s, visible: false }))
    }, 5000)
  }

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
        'backup-id': (config['backup-id'] || '').trim(),
        usevss: config.usevss || false
      }
      await SaveConfig(trimmedConfig)
      setConfig(trimmedConfig)
      showStatus('✅ Configuration enregistrée', 'success')
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
        'backup-id': (config['backup-id'] || '').trim(),
        usevss: config.usevss || false
      }
      await TestConnection(testConfig)
      showStatus('✅ Connexion réussie !', 'success')
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
        showStatus('✅ Configuration chargée depuis le fichier', 'success')
      } catch (err) {
        showStatus('❌ Erreur : fichier JSON invalide', 'error')
      }
    }
    reader.readAsText(file)
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
    showStatus('🚀 Démarrage de la sauvegarde...', 'info')
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
      showStatus('⏳ Sauvegarde en cours...', 'info')
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

    const destPath = prompt('Chemin de destination pour la restauration:', 'C:\\Restore')
    if (!destPath) return

    showStatus(`🔄 Restauration du snapshot ${time}...`, 'info')

    try {
      await RestoreSnapshot(snapshotId, destPath)
      showStatus('✅ Restauration terminée !', 'success')
    } catch (err) {
      showStatus(`❌ ${err}`, 'error')
    }
  }

  return (
    <>
      <div className="header">
        <h1>🛡️ Nimbus Backup</h1>
        <p>Client de sauvegarde pour Proxmox Backup Server - RDEM Systems</p>
      </div>

      <div className="container">
        <div className="tabs">
          <div className={`tab ${activeTab === 'config' ? 'active' : ''}`} onClick={() => setActiveTab('config')}>
            Configuration PBS
          </div>
          <div className={`tab ${activeTab === 'backup' ? 'active' : ''}`} onClick={() => setActiveTab('backup')}>
            Sauvegarde
          </div>
          <div className={`tab ${activeTab === 'restore' ? 'active' : ''}`} onClick={() => setActiveTab('restore')}>
            Restauration
          </div>
          <div className={`tab ${activeTab === 'about' ? 'active' : ''}`} onClick={() => setActiveTab('about')}>
            À propos
          </div>
        </div>

        {/* Config Tab */}
        <div className={`tab-content ${activeTab === 'config' ? 'active' : ''}`}>
          <h2>Configuration du serveur PBS</h2>

          {(config['backup-id'] || hostname) && (
            <div className="info-box" style={{marginBottom: '20px'}}>
              🖥️ <strong>Machine :</strong> {config['backup-id'] || hostname}
            </div>
          )}

          <div className="form-group">
            <label>URL du serveur PBS</label>
            <input
              type="text"
              value={config.baseurl}
              onChange={(e) => setConfig({...config, baseurl: e.target.value})}
              placeholder="https://pbs.example.com:8007"
            />
          </div>

          <div className="form-group">
            <label>Empreinte certificat SSL (optionnel)</label>
            <input
              type="text"
              value={config.certfingerprint}
              onChange={(e) => setConfig({...config, certfingerprint: e.target.value})}
              placeholder="AA:BB:CC:DD:..."
            />
          </div>

          <div className="form-group">
            <label>Authentication ID</label>
            <input
              type="text"
              value={config.authid}
              onChange={(e) => setConfig({...config, authid: e.target.value})}
              placeholder="backup@pbs!token-name"
            />
          </div>

          <div className="form-group">
            <label>Secret (API Token)</label>
            <input
              type="password"
              value={config.secret}
              onChange={(e) => setConfig({...config, secret: e.target.value})}
              placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
            />
          </div>

          <div className="form-group">
            <label>Datastore</label>
            <input
              type="text"
              value={config.datastore}
              onChange={(e) => setConfig({...config, datastore: e.target.value})}
              placeholder="backup-prod"
            />
          </div>

          <div className="form-group">
            <label>Namespace (optionnel)</label>
            <input
              type="text"
              value={config.namespace}
              onChange={(e) => setConfig({...config, namespace: e.target.value})}
              placeholder="production"
            />
          </div>

          <div className="form-group">
            <label>Backup ID (identifiant de sauvegarde)</label>
            <input
              type="text"
              value={config['backup-id']}
              onChange={(e) => setConfig({...config, 'backup-id': e.target.value})}
              placeholder={hostname || "Nom de la machine"}
            />
            <small style={{color: '#718096', fontSize: '12px'}}>
              Laissez vide pour utiliser le nom de machine détecté : {hostname}
            </small>
          </div>

          <div className="form-group">
            <label>Charger une configuration existante</label>
            <input
              type="file"
              id="configFile"
              accept=".json,.txt"
              style={{display: 'none'}}
              onChange={handleLoadConfigFile}
            />
            <button className="btn btn-secondary" onClick={() => document.getElementById('configFile').click()}>
              📁 Charger mon fichier (.json ou .txt)
            </button>
          </div>

          <button className="btn" onClick={handleTestConnection}>Tester la connexion</button>
          <button className="btn btn-secondary" onClick={handleSaveConfig}>Enregistrer</button>

          {/* Upsell message if no backup configured */}
          {!config.baseurl && (
            <div className="info-box" style={{backgroundColor: '#eef2ff', borderLeft: '4px solid #667eea'}}>
              <strong>📦 Vous n'avez pas encore de serveur PBS ?</strong><br/>
              <a
                href={`https://nimbus.rdem-systems.com/choisir-mon-backup/?utm_source=NimbusGui&utm_medium=tooling&utm_campaign=version-${appVersion}&utm_content=config-empty`}
                target="_blank"
                rel="noopener noreferrer"
                style={{color: '#667eea', fontWeight: 'bold', textDecoration: 'underline'}}
              >
                Commander du stockage Nimbus Backup →
              </a>
            </div>
          )}

          <div className="info-box">
            💡 <strong>Astuce :</strong> Obtenez votre API Token depuis l'interface PBS:<br/>
            Configuration → Access Control → API Tokens
          </div>

          {status.visible && activeTab === 'config' && (
            <div className={`status ${status.type} visible`}>{status.message}</div>
          )}
        </div>

        {/* Backup Tab */}
        <div className={`tab-content ${activeTab === 'backup' ? 'active' : ''}`}>
          <h2>Sauvegarde</h2>

          <div className="form-group">
            <label>Type de sauvegarde</label>
            <select value={backupType} onChange={(e) => setBackupType(e.target.value)}>
              <option value="directory">📁 Répertoire (dossier spécifique)</option>
              {/* <option value="machine">💾 Machine (disque complet)</option> */}
            </select>
          </div>

          {/* Backup Mode Toggle */}
          <div className="form-group">
            <label>Mode d'exécution</label>
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
                ⚡ One-shot (maintenant)
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
                📅 Planifié
              </button>
            </div>
          </div>

          {/* Scheduling Options */}
          {backupMode === 'scheduled' && (
            <div className="card" style={{marginTop: '20px', padding: '20px'}}>
              <h3 style={{marginTop: 0}}>⏰ Configuration de la planification</h3>

              {editingJobId && (
                <div className="info-box" style={{backgroundColor: '#fff3cd', borderColor: '#ffc107', marginBottom: '15px'}}>
                  ✏️ <strong>Mode édition</strong> - Modifiez les paramètres et cliquez sur "Mettre à jour"
                </div>
              )}

              <div className="form-group">
                <label>Heure d'exécution quotidienne</label>
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
                  <span>🚀 Exécuter aussi au démarrage de la machine</span>
                </label>
              </div>

              <div className="info-box" style={{backgroundColor: '#eef2ff'}}>
                💡 Le backup sera exécuté automatiquement chaque jour à <strong>{scheduleTime}</strong>
                {runAtStartup && <><br/>Et également à chaque démarrage du système.</>}
              </div>
            </div>
          )}

          {backupType === 'directory' ? (
            <div className="form-group">
              <label>Répertoires à sauvegarder (un par ligne)</label>
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
                <label>Disques physiques à sauvegarder</label>
                {physicalDisks.length === 0 ? (
                  <div style={{padding: '10px', backgroundColor: '#f8f9fa', borderRadius: '4px'}}>
                    🔍 Chargement des disques disponibles...
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
                <label>Fichiers à exclure (un par ligne, optionnel)</label>
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
            <label>Backup ID</label>
            <input
              type="text"
              value={config['backup-id']}
              onChange={(e) => setConfig({...config, 'backup-id': e.target.value})}
              placeholder="Laissez vide pour utiliser le hostname"
            />
          </div>

          <div className="form-group">
            <label>
              <input
                type="checkbox"
                checked={config.usevss}
                onChange={(e) => setConfig({...config, usevss: e.target.checked})}
              />
              Utiliser VSS (Windows Shadow Copy)
            </label>
            {config.usevss && (
              <div className="info-box" style={{marginTop: '10px', backgroundColor: '#fff3cd', borderColor: '#ffc107'}}>
                ⚠️ <strong>VSS nécessite des privilèges administrateur.</strong><br/>
                Redémarrez l'application en tant qu'administrateur (clic droit → Exécuter en tant qu'administrateur) pour utiliser VSS.
              </div>
            )}
          </div>

          {progress > 0 && progress < 100 && (
            <div style={{marginTop: '20px', marginBottom: '20px', padding: '15px', backgroundColor: '#f8f9fa', borderRadius: '8px', border: '1px solid #dee2e6'}}>
              <div style={{display: 'flex', justifyContent: 'space-between', marginBottom: '10px'}}>
                <strong style={{fontSize: '15px'}}>📊 Progression du backup</strong>
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
                    ⏱️ <strong>Temps restant:</strong> {Math.floor(backupStats.eta / 60)}m {backupStats.eta % 60}s
                  </div>
                )}
                {backupStats.speed > 0 && (
                  <div style={{fontSize: '13px', color: '#495057'}}>
                    ⚡ <strong>Vitesse:</strong> {backupStats.speed.toFixed(1)}%/s
                  </div>
                )}
                {backupStats.startTime && (
                  <div style={{fontSize: '13px', color: '#495057'}}>
                    ⏰ <strong>Temps écoulé:</strong> {Math.floor((Date.now() - backupStats.startTime) / 1000)}s
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
              ? (progress > 0 && progress < 100 ? '⏳ Sauvegarde en cours...' : '🚀 Démarrer la sauvegarde')
              : (editingJobId ? '✏️ Mettre à jour la planification' : '💾 Enregistrer la planification')
            }
          </button>
          {backupMode === 'oneshot' && (
            <button className="btn btn-secondary" onClick={() => setProgress(0)} disabled={progress === 0}>Arrêter</button>
          )}
          {backupMode === 'scheduled' && editingJobId && (
            <button className="btn btn-secondary" onClick={() => {
              setEditingJobId(null)
              setScheduleTime('02:00')
              setRunAtStartup(false)
              setBackupDirs('')
              showStatus('✖️ Édition annulée', 'info')
            }}>
              ✖️ Annuler
            </button>
          )}

          {/* Scheduled Jobs List */}
          {backupMode === 'scheduled' && scheduledJobs.length > 0 && (
            <div className="card" style={{marginTop: '30px'}}>
              <h3 style={{marginTop: 0}}>📅 Jobs planifiés</h3>
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
                          setScheduleTime(job.scheduleTime)
                          setRunAtStartup(job.runAtStartup)
                          setBackupDirs(job.backupDirs.join('\n'))
                          setConfig({...config, 'backup-id': job.backupId, usevss: job.useVSS})
                          setBackupType(job.backupType)
                          setExcludeList(job.excludeList.join('\n'))
                          showStatus('✏️ Mode édition - modifiez et sauvegardez', 'info')
                          window.scrollTo({top: 0, behavior: 'smooth'})
                        }}
                      >
                        ✏️ Éditer
                      </button>
                      <button
                        className="btn btn-secondary"
                        style={{padding: '8px 15px', fontSize: '14px'}}
                        onClick={async () => {
                          try {
                            await DeleteScheduledJob(job.id)
                            setScheduledJobs(scheduledJobs.filter(j => j.id !== job.id))
                            showStatus('Job supprimé', 'success')
                            // Cancel edit mode if deleting the job being edited
                            if (editingJobId === job.id) {
                              setEditingJobId(null)
                            }
                          } catch (err) {
                            showStatus(`❌ Erreur: ${err}`, 'error')
                          }
                        }}
                      >
                        🗑️ Supprimer
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
              <h3 style={{marginTop: 0}}>📜 Historique des sauvegardes (dernières 6)</h3>
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
                            showStatus('Configuration chargée, lancez le backup', 'success')
                            window.scrollTo({top: 0, behavior: 'smooth'})
                          }}
                        >
                          🔄 Relancer
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
          <h2>Restauration</h2>

          <div className="form-group">
            <label>Backup ID à restaurer</label>
            <input
              type="text"
              value={restoreBackupId || hostname}
              onChange={(e) => setRestoreBackupId(e.target.value)}
              placeholder={hostname || "hostname ou ID personnalisé"}
            />
          </div>

          <button className="btn" onClick={handleListSnapshots}>📋 Lister les snapshots disponibles</button>

          {showSnapshots && (
            <div style={{marginTop: '20px'}}>
              <h3>Snapshots disponibles</h3>
              <div className="grid">
                {snapshots.length === 0 ? (
                  <p style={{color: '#718096'}}>Aucun snapshot trouvé</p>
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
                        Restaurer
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}

          <div className="info-box" style={{marginTop: '20px'}}>
            💡 <strong>Restauration :</strong> Sélectionnez d'abord un Backup ID, puis listez les snapshots disponibles.<br/>
            Vous pourrez ensuite choisir un snapshot spécifique et le répertoire de destination pour la restauration.
          </div>

          {status.visible && activeTab === 'restore' && (
            <div className={`status ${status.type} visible`}>{status.message}</div>
          )}
        </div>

        {/* About Tab */}
        <div className={`tab-content ${activeTab === 'about' ? 'active' : ''}`}>
          <h2 style={{textAlign: 'center'}}>À propos</h2>

          <img
            src="https://nimbus.rdem-systems.com/logo.webp"
            alt="Nimbus Backup"
            className="logo"
            onError={(e) => e.target.style.display = 'none'}
          />

          <div style={{textAlign: 'center', marginTop: '30px'}}>
            <h3>Nimbus Backup</h3>
            <p style={{color: '#718096', margin: '10px 0'}}>Version {appVersion}</p>

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
                📦 Commander du stockage Nimbus Backup
              </a>
            </div>

            <div className="grid" style={{marginTop: '30px', textAlign: 'left'}}>
              <div className="card">
                <h3>✅ Fonctionnalités</h3>
                <ul style={{lineHeight: 2, marginLeft: '20px'}}>
                  <li>Sauvegarde répertoires & disques</li>
                  <li>Machine complète (C:\, D:\, etc.)</li>
                  <li>Restauration snapshots</li>
                  <li>Support VSS (Shadow Copy)</li>
                  <li>Déduplication & compression</li>
                  <li>Interface Wails moderne</li>
                </ul>
              </div>

              <div className="card">
                <h3>🚀 Technologie</h3>
                <ul style={{lineHeight: 2, marginLeft: '20px'}}>
                  <li>Wails v2 (Go + React)</li>
                  <li>Performance native</li>
                  <li>Interface moderne</li>
                  <li>Logs de debug intégrés</li>
                  <li>Pas de dépendance GPU</li>
                </ul>
              </div>
            </div>

            <p style={{marginTop: '30px'}}>
              <strong>© 2026 RDEM Systems</strong><br/>
              <a href="https://nimbus.rdem-systems.com" style={{color: '#667eea'}}>nimbus.rdem-systems.com</a>
            </p>

            <p style={{marginTop: '20px', color: '#718096', fontSize: '12px'}}>
              Basé sur proxmoxbackupclient_go par tizbac<br/>
              Interface Wails + React + Vite
            </p>
          </div>
        </div>
      </div>
    </>
  )
}

export default App
