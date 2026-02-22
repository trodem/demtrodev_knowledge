# =============================================================================
# SHAREPOINT TOOLKIT – Generic SharePoint Online operations (standalone)
# Query any SharePoint site: lists, items, files, permissions.
# Default site from DM_SPO_SITE_URL env var, overridable per call.
# Safety: Read-only defaults. spo_file_upload writes files to SharePoint.
# Entry point: spo_*
#
# FUNCTIONS
#   spo_sites
#   spo_lists
#   spo_items
#   spo_columns
#   spo_files
#   spo_file_download
#   spo_file_upload
#   spo_permissions
#   spo_search
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
_spo_assert_login
#>
function _spo_assert_login {
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
_spo_invoke -Command "spo site list"
#>
function _spo_invoke {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command
    )

    _spo_assert_login
    $raw = Invoke-Expression "m365 $Command --output json"
    if (-not $raw) { return $null }
    try   { return ($raw | ConvertFrom-Json) }
    catch { throw "Failed to parse m365 CLI output as JSON." }
}

<#
.SYNOPSIS
Resolve the SharePoint site URL from parameter or environment variable.
.PARAMETER SiteUrl
Optional site URL override.
.EXAMPLE
_spo_resolve_site -SiteUrl "https://contoso.sharepoint.com/sites/dev"
#>
function _spo_resolve_site {
    param(
        [Parameter(Mandatory = $false)]
        [string]$SiteUrl = ""
    )

    if (-not [string]::IsNullOrWhiteSpace($SiteUrl)) {
        return $SiteUrl
    }

    $envUrl = [Environment]::GetEnvironmentVariable("DM_SPO_SITE_URL")
    if (-not [string]::IsNullOrWhiteSpace($envUrl)) {
        return $envUrl
    }

    throw "No SharePoint site URL provided. Pass -SiteUrl or set DM_SPO_SITE_URL environment variable."
}

<#
.SYNOPSIS
Ask user for confirmation before a write operation.
.PARAMETER Message
Prompt text.
.EXAMPLE
_spo_confirm -Message "Upload file?"
#>
function _spo_confirm {
    param([Parameter(Mandatory = $true)][string]$Message)
    $answer = Read-Host "$Message [y/N]"
    if ($answer -notin @("y", "Y", "yes", "Yes")) {
        throw "Canceled by user."
    }
}

# -----------------------------------------------------------------------------
# Sites
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List SharePoint sites in the tenant.
.DESCRIPTION
Returns all SharePoint Online sites accessible by the current user.
.EXAMPLE
spo_sites
#>
function spo_sites {
    $sites = _spo_invoke -Command "spo site list"

    $sites | Select-Object Title, Url, Template, StorageUsage |
        Sort-Object Title
}

# -----------------------------------------------------------------------------
# Lists and items
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List all visible lists on a SharePoint site.
.DESCRIPTION
Returns non-hidden GenericLists on the specified site.
.PARAMETER SiteUrl
SharePoint site URL. Defaults to DM_SPO_SITE_URL.
.EXAMPLE
spo_lists -SiteUrl "https://contoso.sharepoint.com/sites/dev"
.EXAMPLE
spo_lists
#>
function spo_lists {
    param(
        [Parameter(Mandatory = $false)]
        [string]$SiteUrl = ""
    )

    $url = _spo_resolve_site -SiteUrl $SiteUrl
    $lists = _spo_invoke -Command "spo list list --webUrl ""$url"""

    $lists | Where-Object {
        $_.Hidden -eq $false -and
        $_.BaseTemplate -eq 100
    } | Select-Object Id, Title, ItemCount | Sort-Object Title
}

<#
.SYNOPSIS
List items in a SharePoint list.
.DESCRIPTION
Returns items from the specified list on the given site.
.PARAMETER ListTitle
Title of the list.
.PARAMETER SiteUrl
SharePoint site URL. Defaults to DM_SPO_SITE_URL.
.EXAMPLE
spo_items -ListTitle "Tasks"
.EXAMPLE
spo_items -ListTitle "Projects" -SiteUrl "https://contoso.sharepoint.com/sites/dev"
#>
function spo_items {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle,

        [Parameter(Mandatory = $false)]
        [string]$SiteUrl = ""
    )

    $url = _spo_resolve_site -SiteUrl $SiteUrl
    $items = _spo_invoke -Command "spo listitem list --webUrl ""$url"" --title ""$ListTitle"""

    $items | Select-Object Id, Title
}

<#
.SYNOPSIS
List user-defined columns of a SharePoint list.
.DESCRIPTION
Returns visible, non-system columns with their type and required status.
.PARAMETER ListTitle
Title of the list.
.PARAMETER SiteUrl
SharePoint site URL. Defaults to DM_SPO_SITE_URL.
.EXAMPLE
spo_columns -ListTitle "Tasks"
#>
function spo_columns {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ListTitle,

        [Parameter(Mandatory = $false)]
        [string]$SiteUrl = ""
    )

    $url = _spo_resolve_site -SiteUrl $SiteUrl
    $fields = _spo_invoke -Command "spo field list --webUrl ""$url"" --listTitle ""$ListTitle"""

    $fields | Where-Object {
        $_.Hidden -eq $false -and
        $_.ReadOnlyField -eq $false -and
        $_.FromBaseType -eq $false
    } | Select-Object Title, InternalName, TypeAsString, Required |
        Sort-Object Title
}

# -----------------------------------------------------------------------------
# Files
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List files in a SharePoint document library folder.
.DESCRIPTION
Returns files from the specified folder path on the site.
.PARAMETER FolderUrl
Server-relative folder URL (e.g. "/Shared Documents/Reports").
.PARAMETER SiteUrl
SharePoint site URL. Defaults to DM_SPO_SITE_URL.
.EXAMPLE
spo_files -FolderUrl "/Shared Documents"
.EXAMPLE
spo_files -FolderUrl "/Shared Documents/Reports" -SiteUrl "https://contoso.sharepoint.com/sites/dev"
#>
function spo_files {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FolderUrl,

        [Parameter(Mandatory = $false)]
        [string]$SiteUrl = ""
    )

    $url = _spo_resolve_site -SiteUrl $SiteUrl
    $files = _spo_invoke -Command "spo file list --webUrl ""$url"" --folder ""$FolderUrl"""

    $files | Select-Object Name, ServerRelativeUrl, Length, TimeLastModified |
        Sort-Object Name
}

<#
.SYNOPSIS
Download a file from SharePoint.
.DESCRIPTION
Downloads a file from the specified server-relative URL to a local path.
.PARAMETER FileUrl
Server-relative file URL (e.g. "/Shared Documents/report.xlsx").
.PARAMETER OutFile
Local output file path. Defaults to filename in current directory.
.PARAMETER SiteUrl
SharePoint site URL. Defaults to DM_SPO_SITE_URL.
.EXAMPLE
spo_file_download -FileUrl "/Shared Documents/report.xlsx"
.EXAMPLE
spo_file_download -FileUrl "/Shared Documents/data.csv" -OutFile "C:\temp\data.csv"
#>
function spo_file_download {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FileUrl,

        [Parameter(Mandatory = $false)]
        [string]$OutFile = "",

        [Parameter(Mandatory = $false)]
        [string]$SiteUrl = ""
    )

    $url = _spo_resolve_site -SiteUrl $SiteUrl

    if ([string]::IsNullOrWhiteSpace($OutFile)) {
        $OutFile = Join-Path (Get-Location).Path ([System.IO.Path]::GetFileName($FileUrl))
    }

    _spo_assert_login
    m365 spo file get --webUrl "$url" --url "$FileUrl" --asFile --path "$OutFile"

    $info = Get-Item -LiteralPath $OutFile
    [pscustomobject]@{
        Status = "downloaded"
        Path   = $info.FullName
        Size   = "{0:N2} KB" -f ($info.Length / 1KB)
    }
}

<#
.SYNOPSIS
Upload a file to a SharePoint document library.
.DESCRIPTION
Uploads a local file to the specified folder on the site.
Requires confirmation unless -Force is used.
.PARAMETER FilePath
Local file path to upload.
.PARAMETER FolderUrl
Server-relative destination folder URL.
.PARAMETER SiteUrl
SharePoint site URL. Defaults to DM_SPO_SITE_URL.
.PARAMETER Force
Skip confirmation prompt.
.EXAMPLE
spo_file_upload -FilePath "C:\data\report.xlsx" -FolderUrl "/Shared Documents"
#>
function spo_file_upload {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,

        [Parameter(Mandatory = $true)]
        [string]$FolderUrl,

        [Parameter(Mandatory = $false)]
        [string]$SiteUrl = "",

        [switch]$Force
    )

    if (-not (Test-Path -LiteralPath $FilePath)) {
        throw "File not found: $FilePath"
    }

    $url = _spo_resolve_site -SiteUrl $SiteUrl
    $fileName = [System.IO.Path]::GetFileName($FilePath)

    if (-not $Force) {
        _spo_confirm -Message "Upload '$fileName' to '$FolderUrl'?"
    }

    _spo_assert_login
    m365 spo file add --webUrl "$url" --folder "$FolderUrl" --path "$FilePath"

    [pscustomobject]@{
        Status     = "uploaded"
        File       = $fileName
        Destination = "$FolderUrl/$fileName"
    }
}

# -----------------------------------------------------------------------------
# Permissions
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show permissions of a SharePoint site.
.DESCRIPTION
Returns role assignments for the specified site.
.PARAMETER SiteUrl
SharePoint site URL. Defaults to DM_SPO_SITE_URL.
.EXAMPLE
spo_permissions
.EXAMPLE
spo_permissions -SiteUrl "https://contoso.sharepoint.com/sites/dev"
#>
function spo_permissions {
    param(
        [Parameter(Mandatory = $false)]
        [string]$SiteUrl = ""
    )

    $url = _spo_resolve_site -SiteUrl $SiteUrl
    $perms = _spo_invoke -Command "spo web roleassignment list --webUrl ""$url"""

    $perms | ForEach-Object {
        [pscustomobject]@{
            PrincipalId   = $_.PrincipalId
            Member        = $_.Member.Title
            RoleBindings  = ($_.RoleDefinitionBindings | ForEach-Object { $_.Name }) -join ", "
        }
    }
}

# -----------------------------------------------------------------------------
# Search
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Search SharePoint content.
.DESCRIPTION
Performs a full-text search across SharePoint Online.
.PARAMETER Query
Search query text.
.PARAMETER Limit
Maximum number of results (default 10).
.EXAMPLE
spo_search -Query "budget report 2025"
.EXAMPLE
spo_search -Query "project plan" -Limit 20
#>
function spo_search {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Query,

        [Parameter(Mandatory = $false)]
        [int]$Limit = 10
    )

    $results = _spo_invoke -Command "spo search --queryText ""$Query"" --rowLimit $Limit --selectProperties ""Title,Path,Author,LastModifiedTime"""

    if (-not $results -or -not $results.PrimaryQueryResult) {
        Write-Output "No results found for '$Query'."
        return
    }

    $rows = $results.PrimaryQueryResult.RelevantResults.Table.Rows
    $rows | ForEach-Object {
        $cells = $_.Cells
        $hash = @{}
        foreach ($cell in $cells) {
            $hash[$cell.Key] = $cell.Value
        }
        [pscustomobject]@{
            Title    = $hash["Title"]
            Path     = $hash["Path"]
            Author   = $hash["Author"]
            Modified = $hash["LastModifiedTime"]
        }
    }
}
