# =============================================================================
# POWER AUTOMATE TOOLKIT – Flow management layer (standalone)
# List, inspect, enable, disable and run Power Automate flows.
# Default environment from DM_PP_ENVIRONMENT env var, overridable per call.
# Safety: flow_enable, flow_disable and flow_run change flow state.
#         All other commands are read-only.
# Entry point: flow_*
#
# FUNCTIONS
#   flow_env_list
#   flow_list
#   flow_info
#   flow_runs
#   flow_enable
#   flow_disable
#   flow_run
#   flow_export
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
_flow_assert_login
#>
function _flow_assert_login {
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
_flow_invoke -Command "flow list"
#>
function _flow_invoke {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command
    )

    _flow_assert_login
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
_flow_resolve_env -Environment "DEV"
#>
function _flow_resolve_env {
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
_flow_env_arg -Environment "DEV"
#>
function _flow_env_arg {
    param([string]$Environment = "")
    if ([string]::IsNullOrWhiteSpace($Environment)) { return "" }
    return " --environmentName ""$Environment"""
}

<#
.SYNOPSIS
Ask user for confirmation.
.PARAMETER Message
Prompt text.
.EXAMPLE
_flow_confirm -Message "Enable flow?"
#>
function _flow_confirm {
    param([Parameter(Mandatory = $true)][string]$Message)
    $answer = Read-Host "$Message [y/N]"
    if ($answer -notin @("y", "Y", "yes", "Yes")) {
        throw "Canceled by user."
    }
}

# -----------------------------------------------------------------------------
# Environment discovery
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List available Power Platform environments.
.DESCRIPTION
Returns all environments accessible by the current user.
Useful to find the environment name for other flow commands.
.EXAMPLE
flow_env_list
#>
function flow_env_list {
    $envs = _flow_invoke -Command "pp environment list"

    $envs | ForEach-Object {
        [pscustomobject]@{
            DisplayName = $_.displayName
            Name        = $_.name
            Type        = $_.properties.environmentType
            State       = $_.properties.provisioningState
            Region      = $_.location
        }
    } | Sort-Object DisplayName
}

# -----------------------------------------------------------------------------
# Flow operations
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List all Power Automate flows.
.DESCRIPTION
Returns flows in the specified environment with their state.
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.PARAMETER AsAdmin
List as admin (see all flows, not just own).
.EXAMPLE
flow_list
.EXAMPLE
flow_list -Environment "DEV"
.EXAMPLE
flow_list -AsAdmin
#>
function flow_list {
    param(
        [Parameter(Mandatory = $false)]
        [string]$Environment = "",

        [switch]$AsAdmin
    )

    $env = _flow_resolve_env -Environment $Environment
    $envArg = _flow_env_arg -Environment $env
    $adminFlag = if ($AsAdmin) { " --asAdmin" } else { "" }

    $flows = _flow_invoke -Command "flow list$envArg$adminFlag"

    $flows | ForEach-Object {
        [pscustomobject]@{
            DisplayName = $_.displayName
            Name        = $_.name
            State       = $_.properties.state
            Created     = $_.properties.createdTime
            Modified    = $_.properties.lastModifiedTime
        }
    } | Sort-Object DisplayName
}

<#
.SYNOPSIS
Show detailed info about a specific flow.
.DESCRIPTION
Returns metadata, triggers and actions for the specified flow.
.PARAMETER Name
Flow name (GUID) or display name.
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.EXAMPLE
flow_info -Name "00000000-0000-0000-0000-000000000000"
#>
function flow_info {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $false)]
        [string]$Environment = ""
    )

    $env = _flow_resolve_env -Environment $Environment
    $envArg = _flow_env_arg -Environment $env

    $flow = _flow_invoke -Command "flow get --name ""$Name""$envArg"

    [pscustomobject]@{
        DisplayName = $flow.displayName
        Name        = $flow.name
        State       = $flow.properties.state
        Created     = $flow.properties.createdTime
        Modified    = $flow.properties.lastModifiedTime
        Creator     = $flow.properties.creator.userId
        Trigger     = $flow.properties.definitionSummary.triggers | ForEach-Object { $_.type }
    }
}

<#
.SYNOPSIS
Show run history of a flow.
.DESCRIPTION
Returns recent runs with status and timestamps.
.PARAMETER Name
Flow name (GUID).
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.EXAMPLE
flow_runs -Name "00000000-0000-0000-0000-000000000000"
#>
function flow_runs {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $false)]
        [string]$Environment = ""
    )

    $env = _flow_resolve_env -Environment $Environment
    $envArg = _flow_env_arg -Environment $env

    $runs = _flow_invoke -Command "flow run list --flowName ""$Name""$envArg"

    $runs | ForEach-Object {
        [pscustomobject]@{
            RunName   = $_.name
            Status    = $_.properties.status
            StartTime = $_.properties.startTime
            EndTime   = $_.properties.endTime
            Trigger   = $_.properties.trigger.name
        }
    }
}

# -----------------------------------------------------------------------------
# State changes
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Enable a Power Automate flow.
.DESCRIPTION
Turns on a flow that is currently disabled.
.PARAMETER Name
Flow name (GUID).
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.EXAMPLE
flow_enable -Name "00000000-0000-0000-0000-000000000000"
#>
function flow_enable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $false)]
        [string]$Environment = ""
    )

    $env = _flow_resolve_env -Environment $Environment
    $envArg = _flow_env_arg -Environment $env

    _flow_assert_login
    m365 flow enable --name "$Name"$envArg

    [pscustomobject]@{
        Status = "enabled"
        Flow   = $Name
    }
}

<#
.SYNOPSIS
Disable a Power Automate flow.
.DESCRIPTION
Turns off a flow. The flow will not run on its triggers.
.PARAMETER Name
Flow name (GUID).
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.EXAMPLE
flow_disable -Name "00000000-0000-0000-0000-000000000000"
#>
function flow_disable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $false)]
        [string]$Environment = ""
    )

    $env = _flow_resolve_env -Environment $Environment
    $envArg = _flow_env_arg -Environment $env

    _flow_assert_login
    m365 flow disable --name "$Name"$envArg

    [pscustomobject]@{
        Status = "disabled"
        Flow   = $Name
    }
}

<#
.SYNOPSIS
Trigger a flow run manually.
.DESCRIPTION
Starts an on-demand execution of the specified flow.
Requires confirmation unless -Force is used.
.PARAMETER Name
Flow name (GUID).
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.PARAMETER Force
Skip confirmation prompt.
.EXAMPLE
flow_run -Name "00000000-0000-0000-0000-000000000000"
#>
function flow_run {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $false)]
        [string]$Environment = "",

        [switch]$Force
    )

    if (-not $Force) {
        _flow_confirm -Message "Run flow '$Name'?"
    }

    $env = _flow_resolve_env -Environment $Environment
    $envArg = _flow_env_arg -Environment $env

    _flow_assert_login
    m365 flow run trigger --name "$Name"$envArg

    [pscustomobject]@{
        Status = "triggered"
        Flow   = $Name
    }
}

# -----------------------------------------------------------------------------
# Export
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Export a flow as a ZIP package.
.DESCRIPTION
Exports the specified flow definition to a local ZIP file.
.PARAMETER Name
Flow name (GUID).
.PARAMETER OutFile
Output file path. Defaults to flow name in current directory.
.PARAMETER Environment
Power Platform environment name. Defaults to DM_PP_ENVIRONMENT.
.EXAMPLE
flow_export -Name "00000000-0000-0000-0000-000000000000"
.EXAMPLE
flow_export -Name "00000000-0000-0000-0000-000000000000" -OutFile "C:\exports\myflow.zip"
#>
function flow_export {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $false)]
        [string]$OutFile = "",

        [Parameter(Mandatory = $false)]
        [string]$Environment = ""
    )

    if ([string]::IsNullOrWhiteSpace($OutFile)) {
        $OutFile = Join-Path (Get-Location).Path "$Name.zip"
    }

    $env = _flow_resolve_env -Environment $Environment
    $envArg = _flow_env_arg -Environment $env

    _flow_assert_login
    m365 flow export --name "$Name" --path "$OutFile"$envArg

    [pscustomobject]@{
        Status = "exported"
        Flow   = $Name
        Path   = $OutFile
    }
}
