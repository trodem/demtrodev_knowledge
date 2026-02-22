# =============================================================================
# TOOLKIT MANAGER TOOLKIT – Toolkit lifecycle management layer (standalone)
# List, inspect, create, scaffold and validate DM toolkits.
# Safety: tk_create and tk_scaffold write new files/functions.
#         All other commands are read-only.
# Entry point: tk_*
#
# FUNCTIONS
#   tk_list
#   tk_count
#   tk_info
#   tk_functions
#   tk_create
#   tk_scaffold
#   tk_validate
#   tk_search
#   tk_open
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Returns the plugins base directory.
.DESCRIPTION
Uses script root to locate the plugins folder.
.EXAMPLE
_tk_plugins_dir
#>
function _tk_plugins_dir {
    return $PSScriptRoot
}

<#
.SYNOPSIS
Returns all toolkit files recursively.
.DESCRIPTION
Scans plugins directory for files matching *_Toolkit.ps1.
.EXAMPLE
_tk_get_files
#>
function _tk_get_files {
    Get-ChildItem -Path (_tk_plugins_dir) -Filter "*_Toolkit.ps1" -Recurse |
        Sort-Object Name
}

<#
.SYNOPSIS
Extracts a human-readable label from a toolkit filename.
.PARAMETER FileName
Toolkit file name (e.g. 3_System_Toolkit.ps1).
.EXAMPLE
_tk_label -FileName "3_System_Toolkit.ps1"
#>
function _tk_label {
    param([Parameter(Mandatory = $true)][string]$FileName)

    $name = [System.IO.Path]::GetFileNameWithoutExtension($FileName)
    if ($name -match '^\d+_') {
        $name = $name -replace '^\d+_', ''
    }
    $name = $name -replace '_Toolkit$', ''
    return ($name -replace '_', ' ')
}

<#
.SYNOPSIS
Extracts public function names from a toolkit file.
.PARAMETER FilePath
Full path to the toolkit .ps1 file.
.EXAMPLE
_tk_extract_functions -FilePath "C:\plugins\Docker_Toolkit.ps1"
#>
function _tk_extract_functions {
    param([Parameter(Mandatory = $true)][string]$FilePath)

    $content = Get-Content -Path $FilePath -Raw
    $results = [regex]::Matches($content, '(?m)^function\s+([a-zA-Z0-9_]+)')
    $public = @()
    foreach ($m in $results) {
        $fn = $m.Groups[1].Value
        if ($fn -notmatch '^_') {
            $public += $fn
        }
    }
    return $public
}

<#
.SYNOPSIS
Derives the common prefix from a list of function names.
.PARAMETER Functions
Array of function names.
.EXAMPLE
_tk_derive_prefix -Functions @("sys_uptime","sys_os")
#>
function _tk_derive_prefix {
    param([string[]]$Functions)

    if (-not $Functions -or $Functions.Count -eq 0) { return "" }
    $first = $Functions[0]
    $idx = $first.IndexOf('_')
    if ($idx -lt 0) { return $first }
    return $first.Substring(0, $idx)
}

<#
.SYNOPSIS
Resolves a toolkit file by label, filename, or prefix.
.PARAMETER Name
Label, filename stem, or prefix to match.
.EXAMPLE
_tk_resolve -Name "System"
#>
function _tk_resolve {
    param([Parameter(Mandatory = $true)][string]$Name)

    $files = _tk_get_files
    $needle = $Name.Trim().ToLower()

    foreach ($f in $files) {
        $label = (_tk_label -FileName $f.Name).ToLower()
        $stem  = [System.IO.Path]::GetFileNameWithoutExtension($f.Name).ToLower()
        $fns   = _tk_extract_functions -FilePath $f.FullName
        $pfx   = (_tk_derive_prefix -Functions $fns).ToLower()

        if ($label -eq $needle -or $stem -eq $needle -or $pfx -eq $needle) {
            return $f
        }
    }

    foreach ($f in $files) {
        $label = (_tk_label -FileName $f.Name).ToLower()
        $stem  = [System.IO.Path]::GetFileNameWithoutExtension($f.Name).ToLower()
        if ($label -like "*$needle*" -or $stem -like "*$needle*") {
            return $f
        }
    }

    throw "Toolkit '$Name' not found. Run tk_list to see available toolkits."
}

# -----------------------------------------------------------------------------
# Discovery
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List all toolkits with summary info.
.DESCRIPTION
Shows label, prefix, function count, and relative path for every toolkit.
.EXAMPLE
tk_list
#>
function tk_list {
    $baseDir = _tk_plugins_dir

    _tk_get_files | ForEach-Object {
        $fns   = _tk_extract_functions -FilePath $_.FullName
        $pfx   = _tk_derive_prefix -Functions $fns
        $label = _tk_label -FileName $_.Name
        $rel   = $_.FullName.Replace($baseDir, '').TrimStart('\', '/')

        [pscustomobject]@{
            Label     = $label
            Prefix    = if ($pfx) { "${pfx}_*" } else { "-" }
            Functions = $fns.Count
            File      = $rel
        }
    }
}

<#
.SYNOPSIS
Count toolkits and functions.
.DESCRIPTION
Without parameters returns total toolkit and function count.
With -Name returns the function count for a specific toolkit.
.PARAMETER Name
Optional toolkit label, filename, or prefix to count functions for.
.EXAMPLE
tk_count
.EXAMPLE
tk_count -Name Docker
.EXAMPLE
tk_count -Name sys
#>
function tk_count {
    param(
        [Parameter(Mandatory = $false)]
        [string]$Name
    )

    if ($Name) {
        $file  = _tk_resolve -Name $Name
        $fns   = _tk_extract_functions -FilePath $file.FullName
        $label = _tk_label -FileName $file.Name

        [pscustomobject]@{
            Toolkit   = $label
            Functions = $fns.Count
        }
    }
    else {
        $files    = _tk_get_files
        $totalFns = 0

        foreach ($f in $files) {
            $fns = _tk_extract_functions -FilePath $f.FullName
            $totalFns += $fns.Count
        }

        [pscustomobject]@{
            Toolkits  = $files.Count
            Functions = $totalFns
        }
    }
}

<#
.SYNOPSIS
Show detailed info about a toolkit.
.DESCRIPTION
Displays header, function list, prefix, path, and line count.
.PARAMETER Name
Toolkit label, filename, or prefix (e.g. "System", "Docker", "dc").
.EXAMPLE
tk_info -Name System
.EXAMPLE
tk_info -Name dc
#>
function tk_info {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $file  = _tk_resolve -Name $Name
    $fns   = _tk_extract_functions -FilePath $file.FullName
    $pfx   = _tk_derive_prefix -Functions $fns
    $label = _tk_label -FileName $file.Name
    $lines = (Get-Content -Path $file.FullName).Count

    $header = @()
    foreach ($line in (Get-Content -Path $file.FullName)) {
        if ($line -match '^#') {
            $header += $line
        }
        elseif ($line.Trim() -eq '') {
            continue
        }
        else {
            break
        }
    }

    [pscustomobject]@{
        Label     = $label
        Prefix    = if ($pfx) { "${pfx}_*" } else { "-" }
        Functions = $fns.Count
        Lines     = $lines
        Path      = $file.FullName
        Header    = ($header -join "`n")
    }
}

<#
.SYNOPSIS
List all functions in a toolkit.
.DESCRIPTION
Returns public function names defined in the specified toolkit file.
.PARAMETER Name
Toolkit label, filename, or prefix.
.EXAMPLE
tk_functions -Name Docker
.EXAMPLE
tk_functions -Name sys
#>
function tk_functions {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $file = _tk_resolve -Name $Name
    $fns  = _tk_extract_functions -FilePath $file.FullName
    $label = _tk_label -FileName $file.Name

    $fns | ForEach-Object {
        $help = Get-Help $_ -ErrorAction SilentlyContinue
        [pscustomobject]@{
            Toolkit  = $label
            Function = $_
            Synopsis = if ($help.Synopsis) { $help.Synopsis } else { "-" }
        }
    }
}

# -----------------------------------------------------------------------------
# Creation
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Create a new toolkit file with standard structure.
.DESCRIPTION
Generates a new *_Toolkit.ps1 file with the standard DM header,
strict mode, and a placeholder function. The file is created in the
plugins root directory.
.PARAMETER Name
Toolkit name (e.g. "Network"). Underscores become spaces in label.
.PARAMETER Prefix
Function prefix (e.g. "net"). All functions will start with prefix_.
.PARAMETER Description
One-line description of the toolkit purpose.
.PARAMETER Safety
Safety classification (default: "Read-only - no destructive operations.").
.EXAMPLE
tk_create -Name "Network" -Prefix "net" -Description "Network diagnostic utilities"
.EXAMPLE
tk_create -Name "Cloud_AWS" -Prefix "aws" -Description "AWS CLI helpers" -Safety "May modify cloud resources."
#>
function tk_create {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [Parameter(Mandatory = $true)]
        [string]$Prefix,

        [Parameter(Mandatory = $false)]
        [string]$Description = "",

        [Parameter(Mandatory = $false)]
        [string]$Safety = "Read-only - no destructive operations."
    )

    $baseDir  = _tk_plugins_dir
    $fileName = "${Name}_Toolkit.ps1"
    $filePath = Join-Path $baseDir $fileName

    if (Test-Path -LiteralPath $filePath) {
        throw "Toolkit file '$fileName' already exists at: $filePath"
    }

    $existing = _tk_get_files
    foreach ($f in $existing) {
        $fns = _tk_extract_functions -FilePath $f.FullName
        $existingPfx = _tk_derive_prefix -Functions $fns
        if ($existingPfx -eq $Prefix) {
            throw "Prefix '${Prefix}_' is already used by toolkit '$($f.Name)'."
        }
    }

    $upperName = ($Name -replace '_', ' ').ToUpper()
    $descLine  = if ($Description) { $Description } else { "Auto-generated toolkit." }
    $firstFn   = "${Prefix}_hello"

    $sb = [System.Text.StringBuilder]::new()
    [void]$sb.AppendLine("# =============================================================================")
    [void]$sb.AppendLine("# $upperName TOOLKIT - $descLine (standalone)")
    [void]$sb.AppendLine("# Safety: $Safety")
    [void]$sb.AppendLine("# Entry point: ${Prefix}_*")
    [void]$sb.AppendLine("#")
    [void]$sb.AppendLine("# FUNCTIONS")
    [void]$sb.AppendLine("#   $firstFn")
    [void]$sb.AppendLine("# =============================================================================")
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine('Set-StrictMode -Version Latest')
    [void]$sb.AppendLine('$ErrorActionPreference = "Stop"')
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("# -----------------------------------------------------------------------------")
    [void]$sb.AppendLine("# Internal helpers")
    [void]$sb.AppendLine("# -----------------------------------------------------------------------------")
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("<#")
    [void]$sb.AppendLine(".SYNOPSIS")
    [void]$sb.AppendLine("Ensure a command is available in PATH.")
    [void]$sb.AppendLine(".PARAMETER Name")
    [void]$sb.AppendLine("Command name to validate.")
    [void]$sb.AppendLine(".EXAMPLE")
    [void]$sb.AppendLine("_assert_command_available -Name docker")
    [void]$sb.AppendLine("#>")
    [void]$sb.AppendLine('function _assert_command_available {')
    [void]$sb.AppendLine('    param([Parameter(Mandatory = $true)][string]$Name)')
    [void]$sb.AppendLine('    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {')
    [void]$sb.AppendLine('        throw "Required command ''$Name'' was not found in PATH."')
    [void]$sb.AppendLine('    }')
    [void]$sb.AppendLine('}')
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("# -----------------------------------------------------------------------------")
    [void]$sb.AppendLine("# Public functions")
    [void]$sb.AppendLine("# -----------------------------------------------------------------------------")
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("<#")
    [void]$sb.AppendLine(".SYNOPSIS")
    [void]$sb.AppendLine("Placeholder function for $upperName toolkit.")
    [void]$sb.AppendLine(".DESCRIPTION")
    [void]$sb.AppendLine("Verifies the toolkit loads correctly. Replace with real logic.")
    [void]$sb.AppendLine(".EXAMPLE")
    [void]$sb.AppendLine("$firstFn")
    [void]$sb.AppendLine("#>")
    [void]$sb.AppendLine("function $firstFn {")
    [void]$sb.AppendLine("    Write-Output `"$upperName toolkit loaded successfully. Prefix: ${Prefix}_*`"")
    [void]$sb.AppendLine("}")

    Set-Content -Path $filePath -Value $sb.ToString() -NoNewline -Encoding UTF8
    Write-Output "Created toolkit: $filePath"
    Write-Output "  Label:  $($Name -replace '_', ' ')"
    Write-Output "  Prefix: ${Prefix}_*"
    Write-Output "  Starter function: $firstFn"
    Write-Output ""
    Write-Output "Next steps:"
    Write-Output "  1. Edit the file to add real functions"
    Write-Output "  2. Use tk_scaffold to generate function templates"
    Write-Output "  3. Run tk_validate to check syntax"
}

<#
.SYNOPSIS
Scaffold a new function into an existing toolkit.
.DESCRIPTION
Generates a function template with comment-based help and appends it
to the specified toolkit file. Also updates the FUNCTIONS index.
.PARAMETER Toolkit
Toolkit label, filename, or prefix.
.PARAMETER FunctionName
Full function name including prefix (e.g. "net_ping").
.PARAMETER Synopsis
One-line synopsis for the function.
.PARAMETER Parameters
Comma-separated list of parameter names (e.g. "Host,Count,Timeout").
.EXAMPLE
tk_scaffold -Toolkit "Network" -FunctionName "net_ping" -Synopsis "Ping a host" -Parameters "Host,Count"
.EXAMPLE
tk_scaffold -Toolkit "sys" -FunctionName "sys_disk_free" -Synopsis "Show free disk space"
#>
function tk_scaffold {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Toolkit,

        [Parameter(Mandatory = $true)]
        [string]$FunctionName,

        [Parameter(Mandatory = $true)]
        [string]$Synopsis,

        [Parameter(Mandatory = $false)]
        [string]$Parameters = ""
    )

    $file = _tk_resolve -Name $Toolkit

    $existingFns = _tk_extract_functions -FilePath $file.FullName
    if ($FunctionName -in $existingFns) {
        throw "Function '$FunctionName' already exists in $($file.Name)."
    }

    $pfx = _tk_derive_prefix -Functions $existingFns
    if ($pfx -and -not $FunctionName.StartsWith("${pfx}_")) {
        Write-Warning "Function name '$FunctionName' does not match toolkit prefix '${pfx}_'."
    }

    $paramBlock = ""
    $paramHelp  = ""
    if ($Parameters) {
        $paramNames = $Parameters -split ',' | ForEach-Object { $_.Trim() } | Where-Object { $_ }
        $paramHelpLines = $paramNames | ForEach-Object { ".PARAMETER $_`n$_ value." }
        $paramHelp = "`n" + ($paramHelpLines -join "`n")
        $paramDefs = $paramNames | ForEach-Object {
            '        [Parameter(Mandatory = $true)][string]$' + $_
        }
        $paramBlock = "`n    param(`n" + ($paramDefs -join ",`n`n") + "`n    )`n"
    }

    $sb = [System.Text.StringBuilder]::new()
    [void]$sb.AppendLine("")
    [void]$sb.AppendLine("<#")
    [void]$sb.AppendLine(".SYNOPSIS")
    [void]$sb.AppendLine($Synopsis)
    [void]$sb.AppendLine(".DESCRIPTION")
    [void]$sb.Append($Synopsis)
    [void]$sb.AppendLine($paramHelp)
    [void]$sb.AppendLine(".EXAMPLE")
    [void]$sb.AppendLine($FunctionName)
    [void]$sb.AppendLine("#>")
    [void]$sb.Append("function $FunctionName {")
    [void]$sb.AppendLine($paramBlock)
    [void]$sb.AppendLine('    throw "Not implemented yet."')
    [void]$sb.AppendLine("}")

    $content = Get-Content -Path $file.FullName -Raw
    if (-not $content.EndsWith("`n")) {
        $content += "`n"
    }
    $content += $sb.ToString()
    Set-Content -Path $file.FullName -Value $content -NoNewline -Encoding UTF8

    $indexContent = Get-Content -Path $file.FullName -Raw
    if ($indexContent -match '(?m)^#\s+FUNCTIONS\s*$') {
        $lines = Get-Content -Path $file.FullName
        $newLines = @()
        $inserted = $false
        $inIndex  = $false
        foreach ($line in $lines) {
            if (-not $inserted -and $line -match '^\#\s+FUNCTIONS\s*$') {
                $inIndex = $true
            }
            if ($inIndex -and -not $inserted -and $line -match '^#\s+=+' -and $newLines.Count -gt 1) {
                $newLines += "#   $FunctionName"
                $inserted = $true
                $inIndex  = $false
            }
            $newLines += $line
        }
        if ($inserted) {
            Set-Content -Path $file.FullName -Value ($newLines -join "`n") -NoNewline -Encoding UTF8
        }
    }

    Write-Output "Scaffolded function '$FunctionName' in $($file.Name)"
    Write-Output "  Edit $($file.FullName) to implement the function body."
}

# -----------------------------------------------------------------------------
# Validation
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Validate PowerShell syntax of a toolkit file.
.DESCRIPTION
Parses the toolkit file as a scriptblock to check for syntax errors.
.PARAMETER Name
Toolkit label, filename, or prefix.
.EXAMPLE
tk_validate -Name Docker
#>
function tk_validate {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $file    = _tk_resolve -Name $Name
    $content = Get-Content -Path $file.FullName -Raw

    try {
        [scriptblock]::Create($content) | Out-Null
        $fns = _tk_extract_functions -FilePath $file.FullName

        $issues = @()
        foreach ($fn in $fns) {
            $help = Get-Help $fn -ErrorAction SilentlyContinue
            if (-not $help.Synopsis -or $help.Synopsis -eq $fn) {
                $issues += "  - $fn : missing .SYNOPSIS"
            }
        }

        Write-Output "OK: $($file.Name) - syntax valid, $($fns.Count) functions."
        if ($issues.Count -gt 0) {
            Write-Output "Warnings (missing documentation):"
            $issues | ForEach-Object { Write-Output $_ }
        }
    }
    catch {
        throw "SYNTAX ERROR in $($file.Name):`n$($_.Exception.Message)"
    }
}

# -----------------------------------------------------------------------------
# Search
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Search for functions across all toolkits.
.DESCRIPTION
Searches function names and synopses for a keyword.
.PARAMETER Query
Search keyword or pattern.
.EXAMPLE
tk_search -Query "docker"
.EXAMPLE
tk_search -Query "list"
#>
function tk_search {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Query
    )

    $results = @()
    foreach ($file in (_tk_get_files)) {
        $fns   = _tk_extract_functions -FilePath $file.FullName
        $label = _tk_label -FileName $file.Name

        foreach ($fn in $fns) {
            $help = Get-Help $fn -ErrorAction SilentlyContinue
            $syn  = if ($help.Synopsis) { $help.Synopsis } else { "" }

            if ($fn -like "*$Query*" -or $syn -like "*$Query*") {
                $results += [pscustomobject]@{
                    Toolkit  = $label
                    Function = $fn
                    Synopsis = if ($syn) { $syn } else { "-" }
                }
            }
        }
    }

    if ($results.Count -eq 0) {
        Write-Output "No functions matching '$Query'."
    }
    else {
        $results
    }
}

# -----------------------------------------------------------------------------
# Editor integration
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Open a toolkit file in VS Code.
.DESCRIPTION
Opens the specified toolkit file in the default VS Code instance.
Falls back to notepad if code is not available.
.PARAMETER Name
Toolkit label, filename, or prefix.
.EXAMPLE
tk_open -Name Docker
.EXAMPLE
tk_open -Name sys
#>
function tk_open {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $file = _tk_resolve -Name $Name

    $editor = Get-Command -Name "code" -ErrorAction SilentlyContinue
    if ($editor) {
        & code $file.FullName
    }
    else {
        $editor = Get-Command -Name "notepad" -ErrorAction SilentlyContinue
        if ($editor) {
            & notepad $file.FullName
        }
        else {
            throw "No editor found. Install VS Code or set a default editor."
        }
    }

    Write-Output "Opened: $($file.FullName)"
}
