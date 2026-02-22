# =============================================================================
# STIBS APP TOOLKIT – Application-level inspection and monitoring (standalone)
# Health checks, logs, migrations and runtime info for the STIBS stack.
# Requires docker and a running STIBS development stack.
# Safety: Read-only — no destructive operations.
# Entry point: stibs_app_*
#
# FUNCTIONS
#   stibs_app_health
#   stibs_app_logs
#   stibs_app_env
#   stibs_app_migrations
#   stibs_app_pending_migrations
#   stibs_app_urls
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers — guards and config
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Ensure a command is available in PATH.
.PARAMETER Name
Command name to validate.
.EXAMPLE
_assert_command_available -Name docker
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

<#
.SYNOPSIS
Read an environment variable with a fallback default.
.PARAMETER Name
Environment variable name.
.PARAMETER Default
Value to return if the variable is unset or empty.
.EXAMPLE
_env_or_default -Name "DM_STIBS_BACKEND_CONTAINER" -Default "docker-backend-1"
#>
function _env_or_default {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Default
    )
    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) { return $Default }
    return $value
}

<#
.SYNOPSIS
Load STIBS application config.
.DESCRIPTION
Builds container names and URLs from environment variables with sensible defaults.
.EXAMPLE
_stibs_app_get_config
#>
function _stibs_app_get_config {
    return [pscustomobject]@{
        BackendContainer  = _env_or_default -Name "DM_STIBS_BACKEND_CONTAINER"  -Default "docker-backend-1"
        FrontendContainer = _env_or_default -Name "DM_STIBS_FRONTEND_CONTAINER" -Default "docker-frontend-1"
        RedisContainer    = _env_or_default -Name "DM_STIBS_REDIS_CONTAINER"    -Default "docker-redis-1"
        DbContainer       = _env_or_default -Name "DM_STIBS_DB_CONTAINER"       -Default "docker-mariadb-1"
        BackendUrl        = _env_or_default -Name "DM_STIBS_BACKEND_URL"        -Default "http://localhost:8000"
        FrontendUrl       = _env_or_default -Name "DM_STIBS_FRONTEND_URL"       -Default "http://localhost:4200"
        DbUser            = _env_or_default -Name "DM_STIBS_DB_USER"            -Default "stibs"
        DbPassword        = _env_or_default -Name "DM_STIBS_DB_PASSWORD"        -Default "stibs"
        DbName            = _env_or_default -Name "DM_STIBS_DB_NAME"            -Default "stibs"
    }
}

<#
.SYNOPSIS
Execute a SQL query in the STIBS MariaDB container.
.DESCRIPTION
Runs the given SQL statement via docker exec against the configured database.
Uses sh -c to avoid PowerShell-to-Docker argument passing issues.
.PARAMETER Sql
SQL statement to execute.
.EXAMPLE
_stibs_app_db_query -Sql "SELECT COUNT(*) FROM migrations;"
#>
function _stibs_app_db_query {
    param([Parameter(Mandatory = $true)][string]$Sql)

    $cfg = _stibs_app_get_config
    $oneLine = ($Sql -replace '\r?\n', ' ').Trim()
    $safeSql = $oneLine.Replace("'", "'\''")
    $shCmd = "mysql --user='$($cfg.DbUser)' --password='$($cfg.DbPassword)' --database='$($cfg.DbName)' --batch --skip-column-names -e '$safeSql'"

    docker exec $($cfg.DbContainer) sh -c "$shCmd"
}

# -----------------------------------------------------------------------------
# Health & Info
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Check health of all STIBS stack services.
.DESCRIPTION
Verifies that each container (backend, frontend, mariadb, redis) is running,
and attempts an HTTP health check on the backend and frontend URLs.
.EXAMPLE
stibs_app_health
#>
function stibs_app_health {
    _assert_command_available -Name docker
    $cfg = _stibs_app_get_config

    $containers = @(
        @{ Name = "backend";  Container = $cfg.BackendContainer  }
        @{ Name = "frontend"; Container = $cfg.FrontendContainer }
        @{ Name = "mariadb";  Container = $cfg.DbContainer       }
        @{ Name = "redis";    Container = $cfg.RedisContainer     }
    )

    $results = @()
    foreach ($svc in $containers) {
        $running = $false
        try {
            $state = docker inspect --format '{{.State.Running}}' $svc.Container 2>$null
            $running = $state -eq 'true'
        } catch { }
        $results += [pscustomobject]@{
            Service   = $svc.Name
            Container = $svc.Container
            Running   = $running
        }
    }

    $endpoints = @(
        @{ Name = "backend";  Url = $cfg.BackendUrl  }
        @{ Name = "frontend"; Url = $cfg.FrontendUrl }
    )

    foreach ($ep in $endpoints) {
        try {
            $resp = Invoke-WebRequest -Uri $ep.Url -Method Head -TimeoutSec 5 -UseBasicParsing -ErrorAction Stop
            $results += [pscustomobject]@{
                Service   = "$($ep.Name) HTTP"
                Container = $ep.Url
                Running   = ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 400)
            }
        } catch {
            $results += [pscustomobject]@{
                Service   = "$($ep.Name) HTTP"
                Container = $ep.Url
                Running   = $false
            }
        }
    }

    return $results
}

<#
.SYNOPSIS
Show recent logs from a STIBS container.
.DESCRIPTION
Tails the last N lines from the specified service container. Defaults to the
backend service.
.PARAMETER Service
Service name: backend, frontend, mariadb or redis (default: backend).
.PARAMETER Lines
Number of log lines to show (default 50).
.EXAMPLE
stibs_app_logs
.EXAMPLE
stibs_app_logs -Service frontend -Lines 100
#>
function stibs_app_logs {
    param(
        [ValidateSet("backend", "frontend", "mariadb", "redis")]
        [string]$Service = "backend",
        [int]$Lines = 50
    )

    _assert_command_available -Name docker
    $cfg = _stibs_app_get_config

    $containerMap = @{
        backend  = $cfg.BackendContainer
        frontend = $cfg.FrontendContainer
        mariadb  = $cfg.DbContainer
        redis    = $cfg.RedisContainer
    }

    $container = $containerMap[$Service]
    docker logs --tail $Lines $container
}

<#
.SYNOPSIS
Show environment variables of the STIBS backend container.
.DESCRIPTION
Retrieves the full environment variable list from the running backend container
via docker inspect.
.EXAMPLE
stibs_app_env
#>
function stibs_app_env {
    _assert_command_available -Name docker
    $cfg = _stibs_app_get_config

    docker inspect $($cfg.BackendContainer) |
        ConvertFrom-Json |
        Select-Object -ExpandProperty Config |
        Select-Object -ExpandProperty Env
}

# -----------------------------------------------------------------------------
# Migrations
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List all executed TypeORM migrations.
.DESCRIPTION
Queries the migrations table to show which migrations have been applied,
ordered by execution timestamp descending.
.EXAMPLE
stibs_app_migrations
#>
function stibs_app_migrations {
    _assert_command_available -Name docker
    _stibs_app_db_query -Sql "SELECT id, timestamp, name FROM migrations ORDER BY timestamp DESC;"
}

<#
.SYNOPSIS
Show pending TypeORM migrations not yet applied.
.DESCRIPTION
Lists migration files present in the backend container that do not appear in
the migrations table. Compares filenames against recorded migration timestamps.
.EXAMPLE
stibs_app_pending_migrations
#>
function stibs_app_pending_migrations {
    _assert_command_available -Name docker
    $cfg = _stibs_app_get_config

    $applied = _stibs_app_db_query -Sql "SELECT timestamp FROM migrations;"
    $appliedSet = @{}
    if ($applied) {
        foreach ($line in ($applied -split "`n")) {
            $ts = $line.Trim()
            if ($ts -ne '') { $appliedSet[$ts] = $true }
        }
    }

    $files = docker exec $($cfg.BackendContainer) sh -c "ls -1 dist/migration/ 2>/dev/null || ls -1 src/migration/ 2>/dev/null || echo ''"
    if (-not $files) { return }

    $pending = @()
    foreach ($f in ($files -split "`n")) {
        $fname = $f.Trim()
        if ($fname -eq '' -or $fname -notmatch '^\d+') { continue }
        $ts = ($fname -split '-')[0] -replace '[^\d]', ''
        if (-not $appliedSet.ContainsKey($ts)) {
            $pending += [pscustomobject]@{
                Timestamp = $ts
                File      = $fname
            }
        }
    }

    if ($pending.Count -eq 0) {
        return [pscustomobject]@{ Status = "All migrations applied" }
    }

    return $pending
}

# -----------------------------------------------------------------------------
# Quick Access
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show URLs and ports for all STIBS stack services.
.DESCRIPTION
Returns a summary of accessible endpoints for the local development stack.
.EXAMPLE
stibs_app_urls
#>
function stibs_app_urls {
    $cfg = _stibs_app_get_config

    return @(
        [pscustomobject]@{ Service = "Backend API";  Url = $cfg.BackendUrl;  Port = 8000  }
        [pscustomobject]@{ Service = "Frontend";     Url = $cfg.FrontendUrl; Port = 4200  }
        [pscustomobject]@{ Service = "MariaDB";      Url = "localhost:13306"; Port = 13306 }
        [pscustomobject]@{ Service = "Redis";         Url = "localhost:16379"; Port = 16379 }
    )
}
