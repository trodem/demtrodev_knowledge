# =============================================================================
# WINGET TOOLKIT – Windows package management layer (standalone)
# Install, update, search and remove software via winget.
# Safety: pkg_install, pkg_update, pkg_update_all and pkg_uninstall modify
#         the system. All other commands are read-only.
# Entry point: pkg_*
#
# FUNCTIONS
#   pkg_list
#   pkg_search
#   pkg_info
#   pkg_install
#   pkg_update
#   pkg_update_all
#   pkg_uninstall
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Ensure winget is available in PATH.
.EXAMPLE
_pkg_assert_winget
#>
function _pkg_assert_winget {
    if (-not (Get-Command -Name "winget" -ErrorAction SilentlyContinue)) {
        throw "winget is not installed or not in PATH. Install App Installer from the Microsoft Store."
    }
}

<#
.SYNOPSIS
Ask user for confirmation before a package operation.
.PARAMETER Message
Prompt text to display.
.EXAMPLE
_pkg_confirm -Message "Install Firefox?"
#>
function _pkg_confirm {
    param([Parameter(Mandatory = $true)][string]$Message)
    $answer = Read-Host "$Message [y/N]"
    if ($answer -notin @("y", "Y", "yes", "Yes")) {
        throw "Canceled by user."
    }
}

# -----------------------------------------------------------------------------
# Read operations
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List installed packages.
.DESCRIPTION
Shows all packages currently installed on the system via winget.
Optionally filter by name.
.PARAMETER Name
Optional name filter.
.EXAMPLE
pkg_list
.EXAMPLE
pkg_list -Name "node"
#>
function pkg_list {
    param(
        [Parameter(Mandatory = $false)]
        [string]$Name = ""
    )

    _pkg_assert_winget

    if ($Name) {
        winget list --name $Name --accept-source-agreements
    }
    else {
        winget list --accept-source-agreements
    }
}

<#
.SYNOPSIS
Search for a package in winget repositories.
.DESCRIPTION
Searches winget sources for packages matching the query.
.PARAMETER Query
Search term.
.EXAMPLE
pkg_search -Query "firefox"
.EXAMPLE
pkg_search -Query "visual studio code"
#>
function pkg_search {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Query
    )

    _pkg_assert_winget
    winget search $Query --accept-source-agreements
}

<#
.SYNOPSIS
Show detailed info about a package.
.DESCRIPTION
Displays metadata for a specific package including version, publisher,
description and install location.
.PARAMETER Id
Package identifier (e.g. Mozilla.Firefox, Microsoft.VisualStudioCode).
.EXAMPLE
pkg_info -Id "Mozilla.Firefox"
#>
function pkg_info {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Id
    )

    _pkg_assert_winget
    winget show --id $Id --accept-source-agreements
}

# -----------------------------------------------------------------------------
# Install / Update
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Install a package via winget.
.DESCRIPTION
Installs the specified package. Requires confirmation unless -Force is used.
.PARAMETER Id
Package identifier (e.g. Mozilla.Firefox).
.PARAMETER Force
Skip confirmation prompt.
.EXAMPLE
pkg_install -Id "Mozilla.Firefox"
.EXAMPLE
pkg_install -Id "Notepad++.Notepad++" -Force
#>
function pkg_install {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Id,

        [switch]$Force
    )

    _pkg_assert_winget

    if (-not $Force) {
        _pkg_confirm -Message "Install package '$Id'?"
    }

    winget install --id $Id --accept-package-agreements --accept-source-agreements
}

<#
.SYNOPSIS
Update a specific package via winget.
.DESCRIPTION
Updates the specified package to the latest available version.
.PARAMETER Id
Package identifier.
.PARAMETER Force
Skip confirmation prompt.
.EXAMPLE
pkg_update -Id "Mozilla.Firefox"
#>
function pkg_update {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Id,

        [switch]$Force
    )

    _pkg_assert_winget

    if (-not $Force) {
        _pkg_confirm -Message "Update package '$Id'?"
    }

    winget upgrade --id $Id --accept-package-agreements --accept-source-agreements
}

<#
.SYNOPSIS
Update all packages with available upgrades.
.DESCRIPTION
Runs winget upgrade --all to update every package that has a newer version.
Requires confirmation unless -Force is used.
.PARAMETER Force
Skip confirmation prompt.
.EXAMPLE
pkg_update_all
.EXAMPLE
pkg_update_all -Force
#>
function pkg_update_all {
    param([switch]$Force)

    _pkg_assert_winget

    if (-not $Force) {
        _pkg_confirm -Message "Update ALL packages?"
    }

    winget upgrade --all --accept-package-agreements --accept-source-agreements
}

# -----------------------------------------------------------------------------
# Uninstall
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Uninstall a package via winget.
.DESCRIPTION
Removes the specified package. Requires confirmation unless -Force is used.
.PARAMETER Id
Package identifier.
.PARAMETER Force
Skip confirmation prompt.
.EXAMPLE
pkg_uninstall -Id "Mozilla.Firefox"
#>
function pkg_uninstall {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Id,

        [switch]$Force
    )

    _pkg_assert_winget

    if (-not $Force) {
        _pkg_confirm -Message "Uninstall package '$Id'?"
    }

    winget uninstall --id $Id
}
