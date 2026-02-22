# =============================================================================
# ARCHIVE TOOLKIT – Compression and extraction layer (standalone)
# Create and extract zip and tar.gz archives.
# Safety: arc_zip and arc_tar create new files. arc_unzip and arc_untar
#         extract files to disk. arc_list is read-only.
# Entry point: arc_*
#
# FUNCTIONS
#   arc_zip
#   arc_unzip
#   arc_list
#   arc_tar
#   arc_untar
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
_assert_command_available -Name tar
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

<#
.SYNOPSIS
Ensure a filesystem path exists.
.PARAMETER Path
Path to validate.
.EXAMPLE
_assert_path_exists -Path "C:\Data"
#>
function _assert_path_exists {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required path '$Path' does not exist."
    }
}

# -----------------------------------------------------------------------------
# Zip
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Create a zip archive from a file or folder.
.DESCRIPTION
Compresses the specified source path into a zip file.
If no output path is given, creates the archive next to the source
with the same name and .zip extension.
.PARAMETER Source
File or folder to compress.
.PARAMETER OutFile
Output zip file path. Defaults to <source>.zip.
.PARAMETER Force
Overwrite existing archive without asking.
.EXAMPLE
arc_zip -Source "C:\Projects\myapp"
.EXAMPLE
arc_zip -Source "C:\Data\report.csv" -OutFile "C:\Backups\report.zip"
#>
function arc_zip {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Source,

        [Parameter(Mandatory = $false)]
        [string]$OutFile = "",

        [switch]$Force
    )

    _assert_path_exists -Path $Source

    if ([string]::IsNullOrWhiteSpace($OutFile)) {
        $baseName = [System.IO.Path]::GetFileNameWithoutExtension($Source)
        $parentDir = Split-Path -Parent $Source
        if ([string]::IsNullOrWhiteSpace($parentDir)) {
            $parentDir = (Get-Location).Path
        }
        $OutFile = Join-Path $parentDir "$baseName.zip"
    }

    if ((Test-Path -LiteralPath $OutFile) -and -not $Force) {
        throw "Archive '$OutFile' already exists. Use -Force to overwrite."
    }

    if (Test-Path -LiteralPath $OutFile) {
        Remove-Item -LiteralPath $OutFile -Force
    }

    Compress-Archive -Path $Source -DestinationPath $OutFile -Force

    $info = Get-Item -LiteralPath $OutFile
    [pscustomobject]@{
        Status  = "created"
        Archive = $info.FullName
        Size    = "{0:N2} KB" -f ($info.Length / 1KB)
    }
}

<#
.SYNOPSIS
Extract a zip archive.
.DESCRIPTION
Extracts the contents of a zip file to the specified destination.
Defaults to a folder with the archive name in the same directory.
.PARAMETER Archive
Path to the zip file.
.PARAMETER Destination
Extraction target folder. Defaults to <archive-name> folder.
.PARAMETER Force
Overwrite existing files without asking.
.EXAMPLE
arc_unzip -Archive "C:\Backups\report.zip"
.EXAMPLE
arc_unzip -Archive "data.zip" -Destination "C:\temp\output"
#>
function arc_unzip {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Archive,

        [Parameter(Mandatory = $false)]
        [string]$Destination = "",

        [switch]$Force
    )

    _assert_path_exists -Path $Archive

    if ([string]::IsNullOrWhiteSpace($Destination)) {
        $baseName = [System.IO.Path]::GetFileNameWithoutExtension($Archive)
        $parentDir = Split-Path -Parent $Archive
        if ([string]::IsNullOrWhiteSpace($parentDir)) {
            $parentDir = (Get-Location).Path
        }
        $Destination = Join-Path $parentDir $baseName
    }

    if ($Force) {
        Expand-Archive -Path $Archive -DestinationPath $Destination -Force
    }
    else {
        Expand-Archive -Path $Archive -DestinationPath $Destination
    }

    $items = Get-ChildItem -Path $Destination -Recurse
    [pscustomobject]@{
        Status      = "extracted"
        Destination = $Destination
        Files       = ($items | Where-Object { -not $_.PSIsContainer }).Count
        Folders     = ($items | Where-Object { $_.PSIsContainer }).Count
    }
}

<#
.SYNOPSIS
List contents of a zip archive.
.DESCRIPTION
Shows all entries inside a zip file without extracting.
.PARAMETER Archive
Path to the zip file.
.EXAMPLE
arc_list -Archive "C:\Backups\report.zip"
#>
function arc_list {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Archive
    )

    _assert_path_exists -Path $Archive

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [System.IO.Compression.ZipFile]::OpenRead($Archive)

    try {
        $totalSize = 0
        $entries = foreach ($entry in $zip.Entries) {
            $totalSize += $entry.Length
            [pscustomobject]@{
                Name          = $entry.FullName
                Size          = "{0:N2} KB" -f ($entry.Length / 1KB)
                CompressedSize = "{0:N2} KB" -f ($entry.CompressedLength / 1KB)
                Modified      = $entry.LastWriteTime.ToString("yyyy-MM-dd HH:mm")
            }
        }

        Write-Output "Archive: $Archive ($($zip.Entries.Count) entries, {0:N2} KB uncompressed)" -f ($totalSize / 1KB)
        $entries
    }
    finally {
        $zip.Dispose()
    }
}

# -----------------------------------------------------------------------------
# Tar
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Create a tar.gz archive from a file or folder.
.DESCRIPTION
Compresses the specified source into a .tar.gz file using the tar command.
Requires tar to be available in PATH (ships with Windows 10+).
.PARAMETER Source
File or folder to compress.
.PARAMETER OutFile
Output file path. Defaults to <source>.tar.gz.
.EXAMPLE
arc_tar -Source "C:\Projects\myapp"
.EXAMPLE
arc_tar -Source "C:\Data" -OutFile "C:\Backups\data.tar.gz"
#>
function arc_tar {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Source,

        [Parameter(Mandatory = $false)]
        [string]$OutFile = ""
    )

    _assert_command_available -Name tar
    _assert_path_exists -Path $Source

    if ([string]::IsNullOrWhiteSpace($OutFile)) {
        $baseName = [System.IO.Path]::GetFileName($Source)
        $parentDir = Split-Path -Parent $Source
        if ([string]::IsNullOrWhiteSpace($parentDir)) {
            $parentDir = (Get-Location).Path
        }
        $OutFile = Join-Path $parentDir "$baseName.tar.gz"
    }

    $resolvedSource = (Resolve-Path -LiteralPath $Source).Path
    $parentOfSource = Split-Path -Parent $resolvedSource
    $entryName = Split-Path -Leaf $resolvedSource

    tar -czf $OutFile -C $parentOfSource $entryName

    $info = Get-Item -LiteralPath $OutFile
    [pscustomobject]@{
        Status  = "created"
        Archive = $info.FullName
        Size    = "{0:N2} KB" -f ($info.Length / 1KB)
    }
}

<#
.SYNOPSIS
Extract a tar.gz archive.
.DESCRIPTION
Extracts the contents of a .tar.gz file to the specified destination.
Requires tar to be available in PATH.
.PARAMETER Archive
Path to the .tar.gz file.
.PARAMETER Destination
Extraction target folder. Defaults to current directory.
.EXAMPLE
arc_untar -Archive "data.tar.gz"
.EXAMPLE
arc_untar -Archive "backup.tar.gz" -Destination "C:\temp"
#>
function arc_untar {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Archive,

        [Parameter(Mandatory = $false)]
        [string]$Destination = ""
    )

    _assert_command_available -Name tar
    _assert_path_exists -Path $Archive

    if ([string]::IsNullOrWhiteSpace($Destination)) {
        $Destination = (Get-Location).Path
    }

    if (-not (Test-Path -LiteralPath $Destination)) {
        New-Item -ItemType Directory -Path $Destination -Force | Out-Null
    }

    tar -xzf $Archive -C $Destination

    [pscustomobject]@{
        Status      = "extracted"
        Archive     = $Archive
        Destination = $Destination
    }
}
