import { useState, useEffect } from 'react'

// Wails runtime imports (will be available when built with Wails)
let GetConfigWithHostname, SaveConfig, TestConnection, StartBackup, ListSnapshots, RestoreSnapshot, ListPhysicalDisks, EventsOn

// Check if we're running in Wails
if (window.go) {
  GetConfigWithHostname = window.go.main.App.GetConfigWithHostname
  SaveConfig = window.go.main.App.SaveConfig
  TestConnection = window.go.main.App.TestConnection
  StartBackup = window.go.main.App.StartBackup
  ListSnapshots = window.go.main.App.ListSnapshots
  RestoreSnapshot = window.go.main.App.RestoreSnapshot
  ListPhysicalDisks = window.go.main.App.ListPhysicalDisks
}

// Wails events
if (window.runtime) {
  EventsOn = window.runtime.EventsOn
}

function App() {
  const [activeTab, setActiveTab] = useState('config')
  const [hostname, setHostname] = useState('')
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

  // Load physical disks when switching to machine mode
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

  // Listen to backup events
  useEffect(() => {
    if (!EventsOn) return

    const unsubProgress = EventsOn('backup:progress', (data) => {
      setProgress(Math.round(data.percent))
      showStatus(`🔄 ${data.message}`, 'info')
    })

    const unsubComplete = EventsOn('backup:complete', (data) => {
      setProgress(data.success ? 100 : 0)
      showStatus(data.success ? '✅ ' + data.message : '❌ ' + data.message, data.success ? 'success' : 'error')
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
      // Trim all string values to remove whitespace
      const trimmedConfig = {
        baseurl: config.baseurl.trim(),
        certfingerprint: config.certfingerprint.trim(),
        authid: config.authid.trim(),
        secret: config.secret.trim(),
        datastore: config.datastore.trim(),
        namespace: config.namespace.trim(),
        backupdir: config.backupdir.trim(),
        'backup-id': config['backup-id'].trim(),
        usevss: config.usevss
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
      await TestConnection()
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
              <option value="machine">💾 Machine (disque complet)</option>
            </select>
          </div>

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

          <div className="progress">
            <div className="progress-bar" style={{width: `${progress}%`}}>{progress}%</div>
          </div>

          <button className="btn" onClick={handleStartBackup}>Démarrer la sauvegarde</button>
          <button className="btn btn-secondary" onClick={() => setProgress(0)}>Arrêter</button>

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
            <p style={{color: '#718096', margin: '10px 0'}}>Version 0.0.16</p>

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
