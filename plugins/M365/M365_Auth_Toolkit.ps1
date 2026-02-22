# =============================================================================
# M365 AUTH TOOLKIT – Microsoft 365 authentication layer (standalone)
# Login, logout and status checks for the m365 CLI.
# Safety: Read-only defaults. m365_login and m365_logout change auth state.
# Entry point: m365_*
#
# FUNCTIONS
#   m365_status
#   m365_login
#   m365_logout
#   m365_user
#   m365_tenant
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
.DESCRIPTION
Checks m365 status and throws if not logged in.
.EXAMPLE
_m365_assert_login
#>
function _m365_assert_login {
    _assert_command_available -Name m365
    $raw = m365 status --output json 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Not authenticated in m365 CLI. Run 'm365_login' first."
    }
    $status = $raw | ConvertFrom-Json
    if ($status.logged -ne $true) {
        throw "Not authenticated in m365 CLI. Run 'm365_login' first."
    }
}

# -----------------------------------------------------------------------------
# Public functions
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show current m365 CLI authentication status.
.DESCRIPTION
Returns whether the user is logged in, the connected user and tenant.
.EXAMPLE
m365_status
#>
function m365_status {
    _assert_command_available -Name m365

    $raw = m365 status --output json 2>$null
    if (-not $raw) {
        [pscustomobject]@{
            Logged   = $false
            Message  = "Not authenticated. Run m365_login."
        }
        return
    }

    $status = $raw | ConvertFrom-Json

    [pscustomobject]@{
        Logged      = $status.logged
        ConnectedAs = $status.connectedAs
        AuthType    = $status.authType
        TenantId    = $status.tenantId
    }
}

<#
.SYNOPSIS
Login to Microsoft 365.
.DESCRIPTION
Starts interactive browser authentication for the m365 CLI.
.EXAMPLE
m365_login
#>
function m365_login {
    _assert_command_available -Name m365
    m365 login
    Write-Output "Login completed. Run m365_status to verify."
}

<#
.SYNOPSIS
Logout from Microsoft 365.
.DESCRIPTION
Clears the current m365 CLI session.
.EXAMPLE
m365_logout
#>
function m365_logout {
    _assert_command_available -Name m365
    m365 logout
    Write-Output "Logged out from Microsoft 365."
}

<#
.SYNOPSIS
Show current authenticated user info.
.DESCRIPTION
Returns display name, email and user principal name of the logged in user.
.EXAMPLE
m365_user
#>
function m365_user {
    _m365_assert_login

    $raw = m365 entra user get --output json 2>$null
    if (-not $raw) {
        throw "Could not retrieve user info."
    }

    $user = $raw | ConvertFrom-Json

    [pscustomobject]@{
        DisplayName       = $user.displayName
        Mail              = $user.mail
        UserPrincipalName = $user.userPrincipalName
        Id                = $user.id
    }
}

<#
.SYNOPSIS
Show tenant information.
.DESCRIPTION
Returns the tenant name, id and default domain.
.EXAMPLE
m365_tenant
#>
function m365_tenant {
    _m365_assert_login

    $raw = m365 tenant id get --output json 2>$null
    if (-not $raw) {
        throw "Could not retrieve tenant info."
    }

    $tenantId = ($raw | ConvertFrom-Json)

    [pscustomobject]@{
        TenantId = $tenantId
    }
}
