Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Change directory to the configured development root.
.DESCRIPTION
Uses $varDevPath from plugins/variables.ps1 and moves the current shell location there.
.EXAMPLE
dm dev_path
#>
function dev_path {
    _assert_path_exists -Path $varDevPath
    Set-Location "$varDevPath"
}

<#
.SYNOPSIS
Change directory to the STIBS docker folder.
.DESCRIPTION
Moves to $varDevPath\50_STIBS\stibs-mono\stibs\docker for local docker compose operations.
.EXAMPLE
dm stibs_path
#>
function stibs_path {
    $target = "$varDevPath\50_STIBS\stibs-mono\stibs\docker"
    _assert_path_exists -Path $target
    Set-Location $target
}

<#
.SYNOPSIS
Change directory to the dm CLI repository.
.DESCRIPTION
Moves to the local knowledge/CLI repository rooted under $varSynologyDrivePath.
.EXAMPLE
dm dm_cli_path
#>
function dm_cli_path {
    $target = "$varSynologyDrivePath\5_Demtrodev_Knowledge"
    _assert_path_exists -Path $target
    Set-Location $target
}

<#
.SYNOPSIS
Change directory to the STIBS mono repository.
.DESCRIPTION
Moves to $varStibsMonoPath to work on the main STIBS project.
.EXAMPLE
dm stibs
#>
function stibs {
    _assert_path_exists -Path $varStibsMonoPath
    Set-Location "$varStibsMonoPath"
}

<#
.SYNOPSIS
Open the STIBS mono repository in VS Code.
.DESCRIPTION
Launches VS Code for $varStibsMonoPath.
.EXAMPLE
dm code-stibs
#>
function code-stibs {
    _assert_command_available -Name code
    _assert_path_exists -Path $varStibsMonoPath
    code "$varStibsMonoPath"
}
