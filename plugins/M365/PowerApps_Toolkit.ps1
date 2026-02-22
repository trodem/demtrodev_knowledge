# =============================================================================
# POWER APPS TOOLKIT – Power Apps management layer (standalone)
# List, inspect, open and export Power Apps.
# Default environment from DM_PP_ENVIRONMENT env var, overridable per call.
# Safety: Read-only defaults. pa_export writes files to disk.
# Entry point: pa_*
#
# FUNCTIONS
#   pa_list
#   pa_info
#   pa_open
#   pa_export
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Ensure a command is available in PATH.
.PARAMETER Name
Command name to validate.
.EXAMPLE
_assert_command_available -Name m365
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH. Install CLI for Microsoft 365: npm i -g @pnp/cli-microsoft365"
    }
}

<#
.SYNOPSIS
Assert that the m365 CLI is authenticated.
.EXAMPLE
_pa_assert_login
#>
function _pa_assert_login {
    _assert_command_available -Name m365
    m365 status 1>$null 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Not authenticated in m365 CLI. Run 'm365_login' or 'm365 login' first."
    }
}

<#
.SYNOPSIS
Execute an m365 CLI command and return parsed JSON.
.PARAMETER Command
Command string without the leading 'm365'.
.EXAMPLE
_pa_invoke -Command "pa app list"
#>
function _pa_invoke {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command
    )

    _pa_assert_login
    $raw = Invoke-Expression "m365 $Command --output json"
    if (-not $raw) { return $null }
    try   { return ($raw | ConvertFrom-Json) }
    catch { throw "Failed to parse m365 CLI output as JSON." }
}

<#
.SYNOPSIS
Resolve Power Platform environment from parameter or env var.
.PARAMETER Environment
Optional environment name or ID override.
.EXAMPLE
_pa_resolve_env -Environment "DEV"
#>
function _pa_resolve_env {
    param(
        [Parameter(Mandatory = $false)]
        [string]$Environment = ""
    )

    if (-not [string]::IsNullOrWhiteSpace($Environment)) {
        return $Environment
    }

    $envVal = [Environment]::GetEnvironmentVariable("DM_PP_ENVIRONMENT")
    if (-not [string]::IsNullOrWhiteSpace($envVal)) {
        return $envVal
    }

    return ""
}

<#
.SYNOPSIS
Build the --environmentName argument string.
.PARAMETER Environment
Resolved environment name or empty.
.EXAMPLE
_pa_env_arg -Environment "DEV"
#>
function _pa_env_arg {
    param([string]$Environment = "")
    if ([string]::IsNullOrWhiteSpace($Environment)) { return "" }
    return " --environmentName ""$Environment"""
}

# -----------------------------------------------------------------------------
# List and inspect
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List all Power Apps.
.DESCRIPTION
Returns apps accessible by the current user in the specified environment.
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.EXAMPLE
pa_list
.EXAMPLE
pa_list -Environment "DEV"
#>
function pa_list {
    param(
        [Parameter(Mandatory = $false)]
        [string]$Environment = ""
    )

    $env = _pa_resolve_env -Environment $Environment
    $envArg = _pa_env_arg -Environment $env

    $apps = _pa_invoke -Command "pa app list$envArg"

    $apps | ForEach-Object {
        [pscustomobject]@{
            DisplayName = $_.displayName
            Name        = $_.name
            Owner       = $_.owner.displayName
            Modified    = $_.lastModifiedTime
            Status      = $_.properties.status
        }
    } | Sort-Object DisplayName
}

<#
.SYNOPSIS
Show detailed info about a Power App.
.DESCRIPTION
Returns metadata including owner, connections and sharing info.
.PARAMETER Name
App name (GUID).
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.EXAMPLE
pa_info -Name "00000000-0000-0000-0000-000000000000"
#>
function pa_info {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $false)]
        [string]$Environment = ""
    )

    $env = _pa_resolve_env -Environment $Environment
    $envArg = _pa_env_arg -Environment $env

    $app = _pa_invoke -Command "pa app get --name ""$Name""$envArg"

    [pscustomobject]@{
        DisplayName = $app.displayName
        Name        = $app.name
        Description = $app.properties.description
        Owner       = $app.owner.displayName
        Created     = $app.properties.createdTime
        Modified    = $app.lastModifiedTime
        Status      = $app.properties.status
        AppType     = $app.properties.appType
        Environment = $app.properties.environment.name
    }
}

# -----------------------------------------------------------------------------
# Actions
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Open a Power App in the browser.
.DESCRIPTION
Launches the app player or maker portal in the default browser.
.PARAMETER Name
App name (GUID).
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.PARAMETER Edit
Open in maker portal (edit mode) instead of player.
.EXAMPLE
pa_open -Name "00000000-0000-0000-0000-000000000000"
.EXAMPLE
pa_open -Name "00000000-0000-0000-0000-000000000000" -Edit
#>
function pa_open {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $false)]
        [string]$Environment = "",

        [switch]$Edit
    )

    $env = _pa_resolve_env -Environment $Environment

    if ($Edit) {
        $envSegment = if ($env) { "&e=$env" } else { "" }
        $url = "https://make.powerapps.com/e/default/app/$Name/edit$envSegment"
    }
    else {
        $url = "https://apps.powerapps.com/play/e/default/$Name"
    }

    Start-Process $url

    [pscustomobject]@{
        Status = "opened"
        App    = $Name
        Mode   = if ($Edit) { "edit" } else { "play" }
        Url    = $url
    }
}

<#
.SYNOPSIS
Export a Power App as a .msapp package.
.DESCRIPTION
Downloads the app package to a local file.
.PARAMETER Name
App name (GUID).
.PARAMETER OutFile
Output file path. Defaults to app name in current directory.
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.EXAMPLE
pa_export -Name "00000000-0000-0000-0000-000000000000"
.EXAMPLE
pa_export -Name "00000000-0000-0000-0000-000000000000" -OutFile "C:\exports\myapp.msapp"
#>
function pa_export {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $false)]
        [string]$OutFile = "",

        [Parameter(Mandatory = $false)]
        [string]$Environment = ""
    )

    if ([string]::IsNullOrWhiteSpace($OutFile)) {
        $OutFile = Join-Path (Get-Location).Path "$Name.msapp"
    }

    $env = _pa_resolve_env -Environment $Environment
    $envArg = _pa_env_arg -Environment $env

    _pa_assert_login
    m365 pa app export --name "$Name" --path "$OutFile"$envArg

    [pscustomobject]@{
        Status = "exported"
        App    = $Name
        Path   = $OutFile
    }
}
