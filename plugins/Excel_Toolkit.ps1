# =============================================================================
# EXCEL TOOLKIT – Auto-generated toolkit (standalone)
# Safety: Review generated functions before use.
# Entry point: excel__*
#
# FUNCTIONS
#   excel_sheets
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
_assert_command_available -Name docker
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
_assert_path_exists -Path C:\Data
#>
function _assert_path_exists {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required path '$Path' does not exist."
    }
}

# -----------------------------------------------------------------------------
# Public functions
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Returns the number of sheets in an Excel file.
.DESCRIPTION
This function takes a file path to an Excel file and returns the count of sheets contained in that file.
.PARAMETER FilePath
The full path to the Excel file.
.EXAMPLE
excel_sheets -FilePath 'C:\Users\Demtro\Downloads\PA.Template{0.0}.xlsx'
#>
function excel_sheets {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath
    )

    _assert_path_exists -Path $FilePath
    
    $excel = [Runtime.Interopservices.Marshal]::GetActiveObject("Excel.Application")
    $workbook = $excel.Workbooks.Open($FilePath)
    $sheetCount = $workbook.Sheets.Count
    $workbook.Close($false)
    return [pscustomobject]@{ SheetCount = $sheetCount }
}

function _assert_path_exists {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )
    if ($null -eq (Test-Path -LiteralPath $Path)) {
        throw "The specified path does not exist: $Path"
    }
}
