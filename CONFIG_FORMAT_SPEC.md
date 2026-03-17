# Formats de Configuration - Proxmox Backup Guardian

Documentation pour générer les fichiers de configuration depuis **members.rdem-systems.com**

## 📋 Format JSON (Recommandé)

### Job complet

```json
{
  "id": "job_1710700800",
  "name": "Backup Serveur Web",
  "description": "Sauvegarde quotidienne du serveur web de production",
  "enabled": true,
  "created": "2026-03-17T10:00:00Z",
  "last_run": "",

  "pbs_config": {
    "baseurl": "https://{{client_pbs_url}}:8007",
    "certfingerprint": "{{pbs_ssl_fingerprint}}",
    "authid": "{{client_authid}}@pbs!{{client_token_name}}",
    "secret": "{{client_api_secret}}",
    "datastore": "{{client_datastore}}",
    "namespace": "{{client_namespace}}",
    "backupdir": "C:\\Data",
    "backup-id": "{{hostname}}",
    "usevss": true
  },

  "folders": [
    "C:\\Data\\WebServer",
    "C:\\Data\\Databases"
  ],

  "disks": [],

  "exclusions": [
    "*.tmp",
    "*.log",
    "node_modules/",
    ".git/"
  ],

  "schedule": "Quotidien (2h du matin)",
  "schedule_cron": "0 2 * * *",

  "keep_last": 7,
  "keep_daily": 14,
  "keep_weekly": 8,
  "keep_monthly": 12,

  "compression": "zstd",
  "chunk_size": "4M",
  "bandwidth_limit": 0,
  "parallel_uploads": 2
}
```

### Configuration PBS minimale

```json
{
  "id": "quick_backup",
  "name": "Backup Simple",
  "enabled": true,

  "pbs_config": {
    "baseurl": "https://nimbus.rdem-systems.com:8007",
    "certfingerprint": "AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99",
    "authid": "client@pbs!backup-token",
    "secret": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "datastore": "client-backup",
    "namespace": "",
    "backupdir": "/home/user/data",
    "backup-id": "",
    "usevss": false
  },

  "folders": ["/home/user/data"],
  "exclusions": [],
  "schedule": "Quotidien (2h du matin)",
  "schedule_cron": "0 2 * * *",
  "keep_last": 7,
  "keep_daily": 14,
  "keep_weekly": 8,
  "keep_monthly": 12
}
```

## 📄 Format INI (Compatible CLI)

### Fichier INI complet

```ini
# Proxmox Backup Guardian Job: Backup Serveur Web
# Generated: 2026-03-17T10:00:00Z
# Client ID: {{client_id}}

[pbs]
baseurl = https://{{client_pbs_url}}:8007
certfingerprint = {{pbs_ssl_fingerprint}}
authid = {{client_authid}}@pbs!{{client_token_name}}
secret = {{client_api_secret}}
datastore = {{client_datastore}}
namespace = {{client_namespace}}

[backup]
# Multi-folders separated by comma
folders = C:\Data\WebServer,C:\Data\Databases
# Or single folder
# backupdir = C:\Data
backup-id = {{hostname}}
usevss = true

[exclusions]
patterns = *.tmp,*.log,node_modules/,.git/,__pycache__/

[schedule]
# Cron format: MIN HOUR DAY MONTH WEEKDAY
cron = 0 2 * * *
# Human readable
description = Quotidien (2h du matin)

[retention]
keep-last = 7
keep-daily = 14
keep-weekly = 8
keep-monthly = 12

[advanced]
compression = zstd
chunk-size = 4M
bandwidth-limit = 0
parallel-uploads = 2

[email]
# Optional email notifications
smtp-host = smtp.example.com
smtp-port = 587
smtp-username = backup@example.com
smtp-password = {{smtp_password}}
email-from = backup@example.com
email-to = admin@example.com
```

## 🔧 Variables à remplacer par members.rdem-systems.com

### Variables client (depuis BDD Laravel)

```php
<?php
// Dans le contrôleur Laravel

$config = [
    // PBS Connection - depuis la souscription du client
    'client_pbs_url' => $subscription->pbs_server_url,  // ex: pbs.rdem-systems.com
    'pbs_ssl_fingerprint' => $subscription->pbs_ssl_fingerprint,
    'client_authid' => $subscription->pbs_username,     // ex: client123
    'client_token_name' => $subscription->pbs_token_name, // ex: backup-token
    'client_api_secret' => $subscription->pbs_api_secret,
    'client_datastore' => $subscription->datastore_name, // ex: client-123-backup
    'client_namespace' => $subscription->namespace ?? '', // ex: production

    // Client info
    'client_id' => $client->id,
    'client_name' => $client->company_name,

    // Backup ID = hostname ou description personnalisée
    // Format: company-slug-hostname ou company-slug-description
    'backup_id' => strtolower(Str::slug($client->company_name)) . '-server-01',
    // OU laisser vide pour auto-détection du hostname
    'backup_id' => '', // Auto: détectera le hostname de la machine

    // Email notifications (optionnel)
    'smtp_password' => $client->smtp_password ?? '',
];

// Générer le JSON
return response()->json($config)->download("backup-config-{$client->id}.json");
```

## 📊 Planifications pré-configurées

### Presets de schedule

```json
{
  "schedules": [
    {
      "name": "Toutes les heures",
      "cron": "0 * * * *",
      "description": "Backup horaire"
    },
    {
      "name": "Toutes les 6 heures",
      "cron": "0 */6 * * *",
      "description": "4 backups par jour"
    },
    {
      "name": "Quotidien (2h du matin)",
      "cron": "0 2 * * *",
      "description": "1 backup par jour à 2h"
    },
    {
      "name": "Hebdomadaire (Dimanche 2h)",
      "cron": "0 2 * * 0",
      "description": "1 backup par semaine"
    },
    {
      "name": "Mensuel (1er jour du mois)",
      "cron": "0 2 1 * *",
      "description": "1 backup par mois"
    }
  ]
}
```

## 🎨 Exemple d'intégration dans Laravel

### Route pour télécharger la config

```php
// routes/web.php
Route::get('/services/backup/download-config', [BackupController::class, 'downloadConfig'])
    ->middleware('auth')
    ->name('backup.download-config');
```

### Contrôleur Laravel

```php
<?php

namespace App\Http\Controllers;

use App\Models\ClientContext;
use Illuminate\Http\Request;

class BackupController extends Controller
{
    public function downloadConfig(Request $request)
    {
        $context = $request->user()->currentContext();
        $subscription = $context->backupSubscription;

        if (!$subscription) {
            return redirect()->back()->with('error', 'Aucun abonnement backup actif');
        }

        // Format: JSON ou INI
        $format = $request->get('format', 'json');

        $config = [
            'id' => 'job_' . time(),
            'name' => "Backup {$context->company_name}",
            'description' => "Sauvegarde automatique générée depuis members.rdem-systems.com",
            'enabled' => true,
            'created' => now()->toIso8601String(),

            'pbs_config' => [
                'baseurl' => "https://{$subscription->pbs_server_url}:8007",
                'certfingerprint' => $subscription->pbs_ssl_fingerprint,
                'authid' => "{$subscription->pbs_username}@pbs!{$subscription->pbs_token_name}",
                'secret' => $subscription->pbs_api_secret,
                'datastore' => $subscription->datastore_name,
                'namespace' => $subscription->namespace ?? '',
                'backupdir' => 'C:\\Data',  // Exemple, à configurer par le client
                'backup-id' => $this->generateBackupId($context),  // Généré automatiquement
                'usevss' => true,
            ],

            'folders' => [],
            'exclusions' => ['*.tmp', '*.log', 'node_modules/', '.git/'],
            'schedule' => 'Quotidien (2h du matin)',
            'schedule_cron' => '0 2 * * *',

            'keep_last' => $subscription->retention_last ?? 7,
            'keep_daily' => $subscription->retention_daily ?? 14,
            'keep_weekly' => $subscription->retention_weekly ?? 8,
            'keep_monthly' => $subscription->retention_monthly ?? 12,

            'compression' => 'zstd',
            'chunk_size' => '4M',
            'bandwidth_limit' => 0,
            'parallel_uploads' => 2,
        ];

        if ($format === 'ini') {
            return $this->generateINI($config, $context->company_name);
        }

        return response()->json($config, 200, [
            'Content-Type' => 'application/json',
            'Content-Disposition' => "attachment; filename=\"backup-{$context->id}.json\"",
        ]);
    }

    private function generateINI(array $config, string $clientName): Response
    {
        $ini = "# Proxmox Backup Guardian - {$clientName}\n";
        $ini .= "# Generated: " . now()->toDateTimeString() . "\n\n";

        $ini .= "[pbs]\n";
        $ini .= "baseurl = {$config['pbs_config']['baseurl']}\n";
        $ini .= "certfingerprint = {$config['pbs_config']['certfingerprint']}\n";
        $ini .= "authid = {$config['pbs_config']['authid']}\n";
        $ini .= "secret = {$config['pbs_config']['secret']}\n";
        $ini .= "datastore = {$config['pbs_config']['datastore']}\n";
        $ini .= "namespace = {$config['pbs_config']['namespace']}\n\n";

        $ini .= "[backup]\n";
        $ini .= "backupdir = {$config['pbs_config']['backupdir']}\n";
        $ini .= "usevss = " . ($config['pbs_config']['usevss'] ? 'true' : 'false') . "\n\n";

        $ini .= "[schedule]\n";
        $ini .= "cron = {$config['schedule_cron']}\n\n";

        $ini .= "[retention]\n";
        $ini .= "keep-last = {$config['keep_last']}\n";
        $ini .= "keep-daily = {$config['keep_daily']}\n";
        $ini .= "keep-weekly = {$config['keep_weekly']}\n";
        $ini .= "keep-monthly = {$config['keep_monthly']}\n";

        return response($ini, 200, [
            'Content-Type' => 'text/plain',
            'Content-Disposition' => "attachment; filename=\"backup-{$clientName}.ini\"",
        ]);
    }

    /**
     * Generate a unique backup-id for the client
     * Format: company-slug-hostname or company-slug-description
     */
    private function generateBackupId(ClientContext $context): string
    {
        $companySlug = Str::slug($context->company_name);

        // Option 1: Laisser vide pour auto-détection du hostname
        // return '';

        // Option 2: Utiliser le nom de l'entreprise comme préfixe
        // Le client ajoutera son hostname dans la GUI: "acme-corp-{hostname}"
        return $companySlug;

        // Option 3: Format personnalisé si le client a plusieurs serveurs
        // return "{$companySlug}-server-01";
    }
}
```

### Vue Blade (bouton de téléchargement)

```blade
{{-- resources/views/services/backup.blade.php --}}

<div class="card">
    <h3>Configuration du client de backup</h3>

    <p>Téléchargez votre fichier de configuration pour le client Proxmox Backup Guardian</p>

    <div class="btn-group">
        <a href="{{ route('backup.download-config', ['format' => 'json']) }}"
           class="btn btn-primary">
            <i class="fas fa-download"></i> Télécharger JSON
        </a>

        <a href="{{ route('backup.download-config', ['format' => 'ini']) }}"
           class="btn btn-secondary">
            <i class="fas fa-download"></i> Télécharger INI
        </a>
    </div>

    <hr>

    <h4>Installation du client</h4>
    <ol>
        <li>Téléchargez le client:
            <a href="{{ asset('downloads/proxmox-backup-gui-windows.exe') }}">Windows</a> |
            <a href="{{ asset('downloads/proxmox-backup-gui-linux') }}">Linux</a>
        </li>
        <li>Installez le client sur votre serveur</li>
        <li>Importez votre fichier de configuration</li>
        <li>Configurez les dossiers à sauvegarder</li>
        <li>Activez la planification automatique</li>
    </ol>
</div>
```

## 🔐 Champs requis vs optionnels

### ✅ Requis

```json
{
  "pbs_config": {
    "baseurl": "REQUIS",
    "authid": "REQUIS",
    "secret": "REQUIS",
    "datastore": "REQUIS"
  }
}
```

### ⚠️ Optionnels (avec valeurs par défaut)

```json
{
  "pbs_config": {
    "certfingerprint": "",  // Peut être vide si certificat valide
    "namespace": "",         // Défaut: racine
    "backup-id": "",         // Défaut: hostname auto
    "usevss": true           // Windows seulement
  },

  "folders": [],             // Configuré par le client dans la GUI
  "exclusions": [],          // Défaut: aucune exclusion

  "schedule_cron": "0 2 * * *",  // Défaut: quotidien 2h

  "keep_last": 7,            // Défaut standard
  "keep_daily": 14,
  "keep_weekly": 8,
  "keep_monthly": 12,

  "compression": "zstd",     // Défaut recommandé
  "chunk_size": "4M",
  "bandwidth_limit": 0,      // 0 = illimité
  "parallel_uploads": 2
}
```

## 📦 Exemple complet pour un client

**Contexte :** Client "ACME Corp" avec abonnement "Drive Bank PBS 2TB"

```json
{
  "id": "job_acme_corp_1710700800",
  "name": "Backup ACME Corp",
  "description": "Sauvegarde générée automatiquement pour ACME Corp - Drive Bank PBS 2TB",
  "enabled": true,
  "created": "2026-03-17T10:00:00Z",

  "pbs_config": {
    "baseurl": "https://pbs-fr-paris.rdem-systems.com:8007",
    "certfingerprint": "5A:3B:8C:... (généré lors de la création du compte)",
    "authid": "acme-corp@pbs!backup-production",
    "secret": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "datastore": "acme-corp-2tb",
    "namespace": "production",
    "backupdir": "",
    "backup-id": "",
    "usevss": true
  },

  "folders": [],
  "exclusions": ["*.tmp", "*.log", "~*", ".DS_Store"],
  "schedule": "Quotidien (2h du matin)",
  "schedule_cron": "0 2 * * *",

  "keep_last": 7,
  "keep_daily": 30,
  "keep_weekly": 12,
  "keep_monthly": 24,

  "compression": "zstd",
  "chunk_size": "4M",
  "bandwidth_limit": 100,
  "parallel_uploads": 4
}
```

---

**Avec ce format, members.rdem-systems.com peut générer automatiquement les configs pour chaque client !** 🎉
