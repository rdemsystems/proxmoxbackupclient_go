import { useState, useEffect } from 'react'

// Wails runtime imports (will be available when built with Wails)
let GetConfig, SaveConfig, TestConnection, StartBackup, ListSnapshots, RestoreSnapshot

// Check if we're running in Wails
if (window.go) {
  GetConfig = window.go.main.App.GetConfig
  SaveConfig = window.go.main.App.SaveConfig
  TestConnection = window.go.main.App.TestConnection
  StartBackup = window.go.main.App.StartBackup
  ListSnapshots = window.go.main.App.ListSnapshots
  RestoreSnapshot = window.go.main.App.RestoreSnapshot
}

function App() {
  const [activeTab, setActiveTab] = useState('config')
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
  const [driveLetter, setDriveLetter] = useState('C:')
  const [excludeList, setExcludeList] = useState('')
  const [progress, setProgress] = useState(0)
  const [status, setStatus] = useState({ message: '', type: '', visible: false })

  const [snapshots, setSnapshots] = useState([])
  const [restoreBackupId, setRestoreBackupId] = useState('')
  const [showSnapshots, setShowSnapshots] = useState(false)

  // Load config on mount
  useEffect(() => {
    if (GetConfig) {
      GetConfig().then(cfg => {
        if (cfg) setConfig(cfg)
      }).catch(err => {
        console.error('Failed to load config:', err)
      })
    }
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
      await SaveConfig(config)
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

    if (backupType === 'directory' && !config.backupdir) {
      showStatus('❌ Répertoire requis', 'error')
      return
    }

    showStatus('🚀 Démarrage de la sauvegarde...', 'info')
    setProgress(30)

    try {
      await StartBackup(
        backupType,
        config.backupdir,
        driveLetter,
        excludeList.split('\n').filter(l => l.trim()),
        config['backup-id'],
        config.usevss
      )
      setProgress(100)
      showStatus('✅ Sauvegarde terminée !', 'success')
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
              <label>Répertoire à sauvegarder</label>
              <input
                type="text"
                value={config.backupdir}
                onChange={(e) => setConfig({...config, backupdir: e.target.value})}
                placeholder="C:\Data ou C:\Users"
              />
            </div>
          ) : (
            <>
              <div className="form-group">
                <label>Disque à sauvegarder</label>
                <select value={driveLetter} onChange={(e) => setDriveLetter(e.target.value)}>
                  <option value="C:">C:\ (Disque système)</option>
                  <option value="D:">D:\</option>
                  <option value="E:">E:\</option>
                  <option value="F:">F:\</option>
                </select>
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
              value={restoreBackupId}
              onChange={(e) => setRestoreBackupId(e.target.value)}
              placeholder="hostname ou ID personnalisé"
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
            <p style={{color: '#718096', margin: '10px 0'}}>Version 0.4.0</p>

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
