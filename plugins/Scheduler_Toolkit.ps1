# =============================================================================
# SCHEDULER TOOLKIT – Windows Task Scheduler management layer (standalone)
# List, inspect, run, enable and disable scheduled tasks.
# Safety: sched_list, sched_info are read-only. sched_run, sched_enable
#         and sched_disable modify task state.
# Entry point: sched_*
#
# FUNCTIONS
#   sched_list
#   sched_info
#   sched_run
#   sched_enable
#   sched_disable
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Resolve a scheduled task by name, with fuzzy matching.
.PARAMETER Name
Full or partial task name.
.EXAMPLE
_sched_resolve -Name "GoogleUpdate"
#>
function _sched_resolve {
    param([Parameter(Mandatory = $true)][string]$Name)

    $exact = Get-ScheduledTask -TaskName $Name -ErrorAction SilentlyContinue
    if ($exact) { return $exact }

    $matches = Get-ScheduledTask | Where-Object { $_.TaskName -like "*$Name*" }
    if ($matches.Count -eq 0) {
        throw "No scheduled task matching '$Name'. Use sched_list to see available tasks."
    }
    if ($matches.Count -gt 5) {
        throw "Too many matches for '$Name' ($($matches.Count)). Be more specific."
    }
    if ($matches.Count -gt 1) {
        $names = ($matches | ForEach-Object { $_.TaskName }) -join ", "
        throw "Multiple matches for '$Name': $names. Be more specific."
    }

    return $matches[0]
}

<#
.SYNOPSIS
Ask user for confirmation.
.PARAMETER Message
Prompt text.
.EXAMPLE
_sched_confirm -Message "Run task now?"
#>
function _sched_confirm {
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
List scheduled tasks.
.DESCRIPTION
Shows scheduled tasks with their state. Optionally filter by name
or restrict to a specific task path. By default shows only root-level
tasks (not Microsoft\Windows internal tasks).
.PARAMETER Name
Optional name filter (partial match).
.PARAMETER Path
Task folder path (default: root "\").
.PARAMETER All
Include all tasks including Microsoft internal ones.
.EXAMPLE
sched_list
.EXAMPLE
sched_list -Name "backup"
.EXAMPLE
sched_list -All
#>
function sched_list {
    param(
        [Parameter(Mandatory = $false)]
        [string]$Name = "",

        [Parameter(Mandatory = $false)]
        [string]$Path = "\",

        [switch]$All
    )

    $tasks = if ($All) {
        Get-ScheduledTask
    }
    else {
        Get-ScheduledTask -TaskPath "$Path" -ErrorAction SilentlyContinue
    }

    if ($Name) {
        $tasks = $tasks | Where-Object { $_.TaskName -like "*$Name*" }
    }

    if (-not $tasks -or $tasks.Count -eq 0) {
        Write-Output "No scheduled tasks found."
        return
    }

    $tasks | Sort-Object TaskName | ForEach-Object {
        $info = $_ | Get-ScheduledTaskInfo -ErrorAction SilentlyContinue
        $lastRun = if ($info -and $info.LastRunTime -and $info.LastRunTime.Year -gt 1999) {
            $info.LastRunTime.ToString("yyyy-MM-dd HH:mm")
        } else { "-" }
        $nextRun = if ($info -and $info.NextRunTime -and $info.NextRunTime.Year -gt 1999) {
            $info.NextRunTime.ToString("yyyy-MM-dd HH:mm")
        } else { "-" }

        [pscustomobject]@{
            Name    = $_.TaskName
            State   = $_.State.ToString()
            LastRun = $lastRun
            NextRun = $nextRun
            Path    = $_.TaskPath
        }
    }
}

<#
.SYNOPSIS
Show detailed info about a scheduled task.
.DESCRIPTION
Displays configuration, triggers, actions and run history for a task.
Supports partial name matching.
.PARAMETER Name
Task name (exact or partial).
.EXAMPLE
sched_info -Name "GoogleUpdate"
#>
function sched_info {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $task = _sched_resolve -Name $Name
    $info = $task | Get-ScheduledTaskInfo -ErrorAction SilentlyContinue

    $triggers = $task.Triggers | ForEach-Object {
        $_.CimClass.CimClassName -replace 'MSFT_Task', '' -replace 'Trigger', ''
    }

    $actions = $task.Actions | ForEach-Object {
        if ($_.Execute) {
            "$($_.Execute) $($_.Arguments)".Trim()
        }
    }

    $lastRun = if ($info -and $info.LastRunTime -and $info.LastRunTime.Year -gt 1999) {
        $info.LastRunTime.ToString("yyyy-MM-dd HH:mm:ss")
    } else { "-" }

    $lastResult = if ($info) { "0x{0:X}" -f $info.LastTaskResult } else { "-" }

    $nextRun = if ($info -and $info.NextRunTime -and $info.NextRunTime.Year -gt 1999) {
        $info.NextRunTime.ToString("yyyy-MM-dd HH:mm:ss")
    } else { "-" }

    [pscustomobject]@{
        Name        = $task.TaskName
        Path        = $task.TaskPath
        State       = $task.State.ToString()
        Description = $task.Description
        Author      = $task.Author
        Triggers    = ($triggers -join ", ")
        Actions     = ($actions -join "; ")
        LastRun     = $lastRun
        LastResult  = $lastResult
        NextRun     = $nextRun
        RunAs       = $task.Principal.UserId
    }
}

# -----------------------------------------------------------------------------
# Modify operations
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Run a scheduled task immediately.
.DESCRIPTION
Triggers an on-demand execution of the specified task.
Requires confirmation unless -Force is used.
.PARAMETER Name
Task name (exact or partial).
.PARAMETER Force
Skip confirmation prompt.
.EXAMPLE
sched_run -Name "MyBackupTask"
.EXAMPLE
sched_run -Name "MyBackupTask" -Force
#>
function sched_run {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,

        [switch]$Force
    )

    $task = _sched_resolve -Name $Name

    if (-not $Force) {
        _sched_confirm -Message "Run task '$($task.TaskName)' now?"
    }

    Start-ScheduledTask -TaskName $task.TaskName -TaskPath $task.TaskPath

    [pscustomobject]@{
        Status = "started"
        Task   = $task.TaskName
    }
}

<#
.SYNOPSIS
Enable a disabled scheduled task.
.DESCRIPTION
Sets the specified task to Enabled state.
.PARAMETER Name
Task name (exact or partial).
.EXAMPLE
sched_enable -Name "MyBackupTask"
#>
function sched_enable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $task = _sched_resolve -Name $Name
    Enable-ScheduledTask -TaskName $task.TaskName -TaskPath $task.TaskPath | Out-Null

    [pscustomobject]@{
        Status = "enabled"
        Task   = $task.TaskName
    }
}

<#
.SYNOPSIS
Disable a scheduled task.
.DESCRIPTION
Sets the specified task to Disabled state. The task remains configured
but will not run on its triggers.
.PARAMETER Name
Task name (exact or partial).
.EXAMPLE
sched_disable -Name "MyBackupTask"
#>
function sched_disable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $task = _sched_resolve -Name $Name
    Disable-ScheduledTask -TaskName $task.TaskName -TaskPath $task.TaskPath | Out-Null

    [pscustomobject]@{
        Status = "disabled"
        Task   = $task.TaskName
    }
}
