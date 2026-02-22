# =============================================================================
# HELP TOOLKIT – AI introspection & discovery layer (standalone)
# Runtime inspection helpers for DM toolkits.
# Safety: Read-only — inspects loaded functions, never modifies state.
# Entry point: help_*
#
# FUNCTIONS
#   help_list_all
#   help_list_prefix
#   help_toolkit_map
#   help_function
#   help_parameters
#   help_examples
#   help_where
#   help_source
#   help_exists
#   help_count
#   help_export_index
#   help_builtin_list
#   help_builtin_info
#   help_overview
#   help_search_intent
#   help_quickref
#   help_env_vars
#   help_prerequisites
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Returns all toolkit functions.
.DESCRIPTION
Returns all functions following toolkit naming convention (prefix_action).
.EXAMPLE
_help_get_toolkit_functions
#>
function _help_get_toolkit_functions {
    Get-Command -CommandType Function |
        Where-Object { $_.Name -match "^[a-zA-Z0-9]+_" }
}

<#
.SYNOPSIS
Extracts prefix from function name.
.DESCRIPTION
Returns prefix portion before first underscore.
.PARAMETER Name
Target function name.
.EXAMPLE
_help_get_prefix -Name "sys_uptime"
#>
function _help_get_prefix {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    return ($Name -split "_")[0]
}

# -----------------------------------------------------------------------------
# Discovery
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List all toolkit functions.
.DESCRIPTION
Returns all loaded toolkit functions sorted by name.
.EXAMPLE
help_list_all
#>
function help_list_all {
    _help_get_toolkit_functions |
        Sort-Object Name |
        Select-Object Name
}

<#
.SYNOPSIS
List toolkit functions by prefix.
.DESCRIPTION
Filters toolkit functions by prefix.
.PARAMETER Prefix
Toolkit prefix.
.EXAMPLE
help_list_prefix -Prefix sys
#>
function help_list_prefix {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Prefix
    )

    _help_get_toolkit_functions |
        Where-Object { $_.Name -like "$Prefix*" } |
        Sort-Object Name |
        Select-Object Name
}

<#
.SYNOPSIS
Group toolkit functions by prefix.
.DESCRIPTION
Returns grouped view of toolkit functions by prefix.
.EXAMPLE
help_toolkit_map
#>
function help_toolkit_map {
    _help_get_toolkit_functions |
        Group-Object { _help_get_prefix $_.Name } |
        Sort-Object Name |
        Select-Object Name, Count
}

<#
.SYNOPSIS
Count toolkit functions.
.DESCRIPTION
Returns total number of toolkit functions loaded.
.EXAMPLE
help_count
#>
function help_count {
    (_help_get_toolkit_functions).Count
}

<#
.SYNOPSIS
Check if toolkit function exists.
.DESCRIPTION
Returns true if function exists and follows toolkit naming convention.
.PARAMETER Name
Target function name.
.EXAMPLE
help_exists -Name sys_uptime
#>
function help_exists {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $cmd = Get-Command -Name $Name -CommandType Function -ErrorAction SilentlyContinue

    if (-not $cmd) { return $false }
    if ($Name -notmatch "^[a-zA-Z0-9]+_") { return $false }

    return $true
}

# -----------------------------------------------------------------------------
# Documentation
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show help for a toolkit function.
.DESCRIPTION
Displays comment-based help for specified toolkit function.
.PARAMETER Name
Target function name.
.EXAMPLE
help_function -Name sys_uptime
#>
function help_function {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    Get-Help $Name -Full
}

<#
.SYNOPSIS
Show parameters of a toolkit function.
.DESCRIPTION
Returns parameter metadata for specified function.
.PARAMETER Name
Target function name.
.EXAMPLE
help_parameters -Name sys_ping
#>
function help_parameters {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    (Get-Command $Name).Parameters.Values |
        Select-Object Name, ParameterType, IsMandatory, Position
}

<#
.SYNOPSIS
Show examples of a toolkit function.
.DESCRIPTION
Displays example section from comment-based help.
.PARAMETER Name
Target function name.
.EXAMPLE
help_examples -Name sys_ping
#>
function help_examples {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    Get-Help $Name -Examples
}

# -----------------------------------------------------------------------------
# Location & Source
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Show where a toolkit function is defined.
.DESCRIPTION
Returns source file or scriptblock information.
.PARAMETER Name
Target function name.
.EXAMPLE
help_where -Name sys_uptime
#>
function help_where {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    $cmd = Get-Command $Name

    return [pscustomobject]@{
        Name       = $cmd.Name
        CommandType = $cmd.CommandType
        Module     = $cmd.ModuleName
        Source     = $cmd.Source
        ScriptPath = $cmd.ScriptBlock.File
    }
}

<#
.SYNOPSIS
Return source code of a toolkit function.
.DESCRIPTION
Outputs the scriptblock definition text.
.PARAMETER Name
Target function name.
.EXAMPLE
help_source -Name sys_uptime
#>
function help_source {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (help_exists -Name $Name)) {
        throw "Function '$Name' not found in toolkit."
    }

    (Get-Command $Name).ScriptBlock.ToString()
}

# -----------------------------------------------------------------------------
# AI support
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Export structured toolkit index.
.DESCRIPTION
Returns structured metadata for all toolkit functions.
Useful for AI reasoning and dynamic discovery.
.EXAMPLE
help_export_index
#>
function help_export_index {

    _help_get_toolkit_functions |
        Sort-Object Name |
        ForEach-Object {
            $help = Get-Help $_.Name -ErrorAction SilentlyContinue

            [pscustomobject]@{
                Name       = $_.Name
                Prefix     = _help_get_prefix $_.Name
                Module     = $_.ModuleName
                Parameters = ($_.Parameters.Keys -join ", ")
                Synopsis   = $help.Synopsis
                ScriptPath = $_.ScriptBlock.File
            }
        }
}

# -----------------------------------------------------------------------------
# Built-in commands
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
List built-in PowerShell commands.
.DESCRIPTION
Returns built-in cmdlets, aliases and module functions.
.PARAMETER Name
Optional name filter (default: all).
.EXAMPLE
help_builtin_list
.EXAMPLE
help_builtin_list -Name "Get-*"
#>
function help_builtin_list {
    param([string]$Name = "*")

    Get-Command -Name $Name |
        Where-Object { $null -ne $_.ModuleName } |
        Sort-Object Name |
        Select-Object Name, CommandType, ModuleName
}

<#
.SYNOPSIS
Show detailed information about a built-in command.
.DESCRIPTION
Returns metadata and help content for specified built-in command.
.PARAMETER Name
Command name.
.EXAMPLE
help_builtin_info -Name Get-Process
#>
function help_builtin_info {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $cmd = Get-Command -Name $Name -ErrorAction Stop

    if ($null -eq $cmd.ModuleName) {
        throw "Command '$Name' is not a built-in module command."
    }

    $help = Get-Help $Name -Full -ErrorAction SilentlyContinue

    return [pscustomobject]@{
        Name       = $cmd.Name
        CommandType = $cmd.CommandType
        Module     = $cmd.ModuleName
        Version    = $cmd.Version
        Source     = $cmd.Source
        Parameters = ($cmd.Parameters.Keys -join ", ")
        Synopsis   = $help.Synopsis
        Syntax     = ($help.Syntax | Out-String).Trim()
    }
}

# -----------------------------------------------------------------------------
# Overview & Discovery (enhanced)
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Complete overview of all toolkits with descriptions, functions and prerequisites.
.DESCRIPTION
Scans all loaded toolkit .ps1 files and returns a structured directory.
Each toolkit entry includes: name, description, prefix, function count,
function list, safety level, required external tools, and environment variables.
Use this to answer questions like "what can I do?", "what toolkits are there?",
"show me everything available".
.PARAMETER Prefix
Optional. Filter overview to a single toolkit prefix (e.g. "spo", "arc").
.EXAMPLE
help_overview
.EXAMPLE
help_overview -Prefix spo
#>
function help_overview {
    param([string]$Prefix)

    $tkFiles = @()
    $pluginsDir = $PSScriptRoot
    if (Test-Path $pluginsDir) {
        $tkFiles = Get-ChildItem -Path $pluginsDir -Filter "*_Toolkit.ps1" -Recurse -File
    }

    $allFunctions = _help_get_toolkit_functions

    foreach ($file in $tkFiles) {
        $headerLines = Get-Content -Path $file.FullName -TotalCount 40 -ErrorAction SilentlyContinue
        $header = $headerLines -join "`n"

        $descLine = ($headerLines | Where-Object { $_ -match "^#\s+\w+" -and $_ -notmatch "^# =+" -and $_ -notmatch "^# FUNCTIONS" -and $_ -notmatch "^# Entry point:" -and $_ -notmatch "^# Safety:" } | Select-Object -First 1)
        $desc = if ($descLine) { ($descLine -replace "^#\s*", "").Trim() } else { "" }

        $safetyLine = ($headerLines | Where-Object { $_ -match "^# Safety:" } | Select-Object -First 1)
        $safety = if ($safetyLine) { ($safetyLine -replace "^# Safety:\s*", "").Trim() } else { "Unknown" }

        $entryLine = ($headerLines | Where-Object { $_ -match "^# Entry point:" } | Select-Object -First 1)
        $entryPrefix = ""
        if ($entryLine -match "Entry point:\s*(\w+)") {
            $entryPrefix = ($Matches[1] -replace "[_*]", "")
        }

        if ($Prefix -and $entryPrefix -and $entryPrefix -ne $Prefix) { continue }

        $tkFunctions = @()
        if ($entryPrefix) {
            $tkFunctions = $allFunctions | Where-Object { $_.Name -like "${entryPrefix}_*" }
        }

        $funcNames = @($tkFunctions | ForEach-Object {
            $h = Get-Help $_.Name -ErrorAction SilentlyContinue
            $syn = if ($h -and $h.Synopsis) { $h.Synopsis.Trim() } else { "" }
            if ($syn) { "$($_.Name) - $syn" } else { $_.Name }
        })

        $envVars = @()
        $content = Get-Content -Path $file.FullName -Raw -ErrorAction SilentlyContinue
        $envMatches = [regex]::Matches($content, 'DM_[A-Z_]+')
        if ($envMatches.Count -gt 0) {
            $envVars = @($envMatches | ForEach-Object { $_.Value } | Sort-Object -Unique)
        }

        $prereqs = @()
        if ($content -match '\bm365\b') { $prereqs += "m365 CLI" }
        if ($content -match '\bdocker\b') { $prereqs += "Docker" }
        if ($content -match '\bwinget\b') { $prereqs += "winget" }
        if ($content -match '\bgit\b') { $prereqs += "git" }

        $relativePath = $file.FullName
        if ($file.FullName.StartsWith($pluginsDir)) {
            $relativePath = $file.FullName.Substring($pluginsDir.Length).TrimStart('\', '/')
        }

        [pscustomobject]@{
            Toolkit       = ($file.BaseName -replace "_Toolkit$", "" -replace "^\d+_", "")
            Description   = $desc
            Prefix        = $entryPrefix
            Safety        = $safety
            FunctionCount = $funcNames.Count
            Functions     = ($funcNames -join "; ")
            EnvVars       = ($envVars -join ", ")
            Prerequisites = ($prereqs -join ", ")
            File          = $relativePath
        }
    }
}

<#
.SYNOPSIS
Search toolkit functions by intent or keyword.
.DESCRIPTION
Searches across function names, synopses and descriptions for the given
keyword or phrase. Useful when the user knows WHAT they want to do but
not WHICH function does it. Returns matching functions ranked by relevance.
Use this to answer "how do I compress?", "is there a download function?",
"what can I do with SharePoint?".
.PARAMETER Query
The search term or phrase (e.g. "compress", "download", "SharePoint list").
.PARAMETER MaxResults
Maximum number of results to return (default 15).
.EXAMPLE
help_search_intent -Query "compress"
.EXAMPLE
help_search_intent -Query "SharePoint list"
.EXAMPLE
help_search_intent -Query "download file"
#>
function help_search_intent {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Query,

        [int]$MaxResults = 15
    )

    $keywords = $Query.ToLower() -split "\s+" | Where-Object { $_.Length -gt 1 }
    if ($keywords.Count -eq 0) {
        throw "Query too short. Provide at least one meaningful keyword."
    }

    $functions = _help_get_toolkit_functions
    $results = @()

    foreach ($fn in $functions) {
        $name = $fn.Name.ToLower()
        $help = Get-Help $fn.Name -ErrorAction SilentlyContinue
        $synopsis = if ($help -and $help.Synopsis) { $help.Synopsis.ToLower() } else { "" }
        $description = ""
        if ($help -and $help.Description) {
            $description = ($help.Description | Out-String).ToLower()
        }

        $score = 0
        foreach ($kw in $keywords) {
            if ($name -match [regex]::Escape($kw)) { $score += 3 }
            if ($synopsis -match [regex]::Escape($kw)) { $score += 2 }
            if ($description -match [regex]::Escape($kw)) { $score += 1 }
        }

        if ($score -gt 0) {
            $params = @()
            try {
                $params = @((Get-Command $fn.Name).Parameters.Values |
                    Where-Object { $_.Name -notin @("Verbose","Debug","ErrorAction","WarningAction","InformationAction","ErrorVariable","WarningVariable","InformationVariable","OutVariable","OutBuffer","PipelineVariable") } |
                    ForEach-Object {
                        $req = if ($_.Attributes | Where-Object { $_ -is [System.Management.Automation.ParameterAttribute] -and $_.Mandatory }) { " (required)" } else { "" }
                        "$($_.Name)$req"
                    })
            } catch {}

            $results += [pscustomobject]@{
                Function    = $fn.Name
                Synopsis    = if ($help -and $help.Synopsis) { $help.Synopsis.Trim() } else { "" }
                Parameters  = ($params -join ", ")
                Relevance   = $score
            }
        }
    }

    $results | Sort-Object -Property Relevance -Descending | Select-Object -First $MaxResults
}

<#
.SYNOPSIS
Quick reference card for a specific toolkit with practical examples.
.DESCRIPTION
Shows a formatted guide for the specified toolkit prefix, including:
all functions with synopsis, parameters, usage examples, and tips.
Designed to give the user everything they need to start using a toolkit.
.PARAMETER Prefix
The toolkit prefix (e.g. "spo", "arc", "sys", "flow").
.EXAMPLE
help_quickref -Prefix arc
.EXAMPLE
help_quickref -Prefix spo
#>
function help_quickref {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Prefix
    )

    $functions = _help_get_toolkit_functions | Where-Object { $_.Name -like "${Prefix}_*" } | Sort-Object Name
    if ($functions.Count -eq 0) {
        throw "No functions found with prefix '$Prefix'. Use help_toolkit_map to see available prefixes."
    }

    foreach ($fn in $functions) {
        $help = Get-Help $fn.Name -Full -ErrorAction SilentlyContinue

        $synopsis = if ($help -and $help.Synopsis) { $help.Synopsis.Trim() } else { "No description" }

        $params = @()
        try {
            $params = @((Get-Command $fn.Name).Parameters.Values |
                Where-Object { $_.Name -notin @("Verbose","Debug","ErrorAction","WarningAction","InformationAction","ErrorVariable","WarningVariable","InformationVariable","OutVariable","OutBuffer","PipelineVariable") } |
                ForEach-Object {
                    $mandatory = ($_.Attributes | Where-Object { $_ -is [System.Management.Automation.ParameterAttribute] -and $_.Mandatory }) -ne $null
                    $type = $_.ParameterType.Name
                    $tag = if ($mandatory) { " (required)" } else { "" }
                    "-$($_.Name) [$type]$tag"
                })
        } catch {}

        $examples = @()
        if ($help -and $help.Examples -and $help.Examples.Example) {
            foreach ($ex in $help.Examples.Example) {
                $code = ($ex.Code | Out-String).Trim()
                if ($code) { $examples += $code }
            }
        }

        [pscustomobject]@{
            Function   = $fn.Name
            Synopsis   = $synopsis
            Parameters = if ($params.Count -gt 0) { $params -join "; " } else { "(none)" }
            Examples   = if ($examples.Count -gt 0) { $examples -join " | " } else { "(none in help)" }
        }
    }
}

<#
.SYNOPSIS
List all environment variables used across DM toolkits.
.DESCRIPTION
Scans all toolkit .ps1 files for DM_* environment variable references.
Returns the variable name, default value (if found), and which toolkit uses it.
Use this to answer "what env vars do I need to set?", "how do I configure
SharePoint?", "what settings are available?".
.PARAMETER Prefix
Optional. Filter to variables used by a specific toolkit prefix.
.EXAMPLE
help_env_vars
.EXAMPLE
help_env_vars -Prefix spo
#>
function help_env_vars {
    param([string]$Prefix)

    $pluginsDir = $PSScriptRoot
    $tkFiles = @()
    if (Test-Path $pluginsDir) {
        $tkFiles = Get-ChildItem -Path $pluginsDir -Filter "*_Toolkit.ps1" -Recurse -File
    }

    $results = @()

    foreach ($file in $tkFiles) {
        $headerLines = Get-Content -Path $file.FullName -TotalCount 20 -ErrorAction SilentlyContinue
        $entryLine = ($headerLines | Where-Object { $_ -match "^# Entry point:" } | Select-Object -First 1)
        $entryPrefix = ""
        if ($entryLine -match "Entry point:\s*(\w+)") {
            $entryPrefix = ($Matches[1] -replace "[_*]", "")
        }

        if ($Prefix -and $entryPrefix -and $entryPrefix -ne $Prefix) { continue }

        $content = Get-Content -Path $file.FullName -Raw -ErrorAction SilentlyContinue
        $tkName = ($file.BaseName -replace "_Toolkit$", "" -replace "^\d+_", "")

        $envMatches = [regex]::Matches($content, 'DM_[A-Z_]+')
        $seen = @{}
        foreach ($m in $envMatches) {
            $varName = $m.Value
            if ($seen.ContainsKey($varName)) { continue }
            $seen[$varName] = $true

            $defaultVal = ""
            if ($content -match [regex]::Escape($varName) + '.*-Default\s+"([^"]*)"') {
                $defaultVal = $Matches[1]
            }

            $currentVal = [Environment]::GetEnvironmentVariable($varName)
            $status = if ($currentVal) { "SET" } else { "NOT SET" }

            $results += [pscustomobject]@{
                Variable   = $varName
                Toolkit    = $tkName
                Prefix     = $entryPrefix
                Default    = $defaultVal
                Status     = $status
                Current    = if ($currentVal) { $currentVal } else { "" }
            }
        }
    }

    $results | Sort-Object Variable
}

<#
.SYNOPSIS
Check which external tools are required by toolkits and if they are installed.
.DESCRIPTION
Scans toolkit files for references to external CLI tools (m365, docker, winget,
git, etc.) and checks if each is available in PATH. Returns a status report.
Use this to answer "what do I need to install?", "is m365 CLI available?",
"why is SharePoint not working?".
.EXAMPLE
help_prerequisites
#>
function help_prerequisites {

    $pluginsDir = $PSScriptRoot
    $tkFiles = @()
    if (Test-Path $pluginsDir) {
        $tkFiles = Get-ChildItem -Path $pluginsDir -Filter "*_Toolkit.ps1" -Recurse -File
    }

    $toolUsage = @{}

    foreach ($file in $tkFiles) {
        $content = Get-Content -Path $file.FullName -Raw -ErrorAction SilentlyContinue
        $tkName = ($file.BaseName -replace "_Toolkit$", "" -replace "^\d+_", "")

        $checks = @(
            @{ Tool = "m365"; Pattern = '\bm365\s' },
            @{ Tool = "docker"; Pattern = '\bdocker\s' },
            @{ Tool = "docker-compose"; Pattern = '\bdocker-compose\b|docker\s+compose\b' },
            @{ Tool = "winget"; Pattern = '\bwinget\s' },
            @{ Tool = "git"; Pattern = '\bgit\s' },
            @{ Tool = "tar"; Pattern = '\btar\s' }
        )

        foreach ($check in $checks) {
            if ($content -match $check.Pattern) {
                if (-not $toolUsage.ContainsKey($check.Tool)) {
                    $toolUsage[$check.Tool] = @()
                }
                $toolUsage[$check.Tool] += $tkName
            }
        }
    }

    foreach ($tool in ($toolUsage.Keys | Sort-Object)) {
        $available = $null -ne (Get-Command $tool -ErrorAction SilentlyContinue)
        $version = ""
        if ($available) {
            try {
                $version = (& $tool --version 2>&1 | Select-Object -First 1).ToString().Trim()
            } catch {
                $version = "(installed)"
            }
        }

        [pscustomobject]@{
            Tool      = $tool
            Installed = $available
            Version   = if ($available) { $version } else { "NOT FOUND" }
            UsedBy    = ($toolUsage[$tool] -join ", ")
        }
    }
}
