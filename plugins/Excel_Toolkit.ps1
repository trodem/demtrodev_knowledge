# =============================================================================
# EXCEL TOOLKIT – Excel spreadsheet operations (standalone)
# Safety: Read-only — no destructive operations.
# Entry point: xls_*
#
# FUNCTIONS
#   xls_sheets       — List all sheet names in an xlsx file
#   xls_info         — Show file metadata (size, sheets, dates)
#   xls_preview      — Preview first N rows of a sheet
#   xls_to_csv       — Export a sheet to CSV
#   xls_search       — Search for a text value across all sheets
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.IO.Compression.FileSystem

# -----------------------------------------------------------------------------
# Internal helpers
# -----------------------------------------------------------------------------

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

<#
.SYNOPSIS
Open an xlsx file as a ZipArchive for reading.
.PARAMETER FilePath
Path to the xlsx file.
.EXAMPLE
$zip = _xls_open_zip -FilePath "C:\data\report.xlsx"
#>
function _xls_open_zip {
    param([Parameter(Mandatory = $true)][string]$FilePath)
    $full = (Resolve-Path -LiteralPath $FilePath).Path
    return [System.IO.Compression.ZipFile]::OpenRead($full)
}

<#
.SYNOPSIS
Read XML content from a zip entry by path.
.PARAMETER Zip
The ZipArchive object.
.PARAMETER EntryPath
Relative path inside the zip.
.EXAMPLE
$xml = _xls_read_entry -Zip $zip -EntryPath "xl/workbook.xml"
#>
function _xls_read_entry {
    param(
        [Parameter(Mandatory = $true)]
        [System.IO.Compression.ZipArchive]$Zip,

        [Parameter(Mandatory = $true)]
        [string]$EntryPath
    )
    $entry = $Zip.GetEntry($EntryPath)
    if ($null -eq $entry) {
        $entry = $Zip.Entries | Where-Object { $_.FullName -eq $EntryPath } | Select-Object -First 1
    }
    if ($null -eq $entry) { throw "Entry '$EntryPath' not found in archive." }
    $stream = $entry.Open()
    try {
        $reader = New-Object System.IO.StreamReader($stream)
        $text = $reader.ReadToEnd()
        $reader.Close()
    } finally {
        $stream.Close()
    }
    [xml]$text
}

<#
.SYNOPSIS
Extract sheet names and their rId from workbook.xml.
.PARAMETER Zip
The ZipArchive object.
.EXAMPLE
$sheets = _xls_sheet_list -Zip $zip
#>
function _xls_sheet_list {
    param([Parameter(Mandatory = $true)][System.IO.Compression.ZipArchive]$Zip)
    $wb = _xls_read_entry -Zip $Zip -EntryPath "xl/workbook.xml"
    $ns = New-Object System.Xml.XmlNamespaceManager($wb.NameTable)
    $ns.AddNamespace("s", "http://schemas.openxmlformats.org/spreadsheetml/2006/main")
    $ns.AddNamespace("r", "http://schemas.openxmlformats.org/officeDocument/2006/relationships")
    $nodes = $wb.SelectNodes("//s:sheet", $ns)
    $rels = _xls_read_entry -Zip $Zip -EntryPath "xl/_rels/workbook.xml.rels"
    $rns = New-Object System.Xml.XmlNamespaceManager($rels.NameTable)
    $rns.AddNamespace("r", "http://schemas.openxmlformats.org/package/2006/relationships")
    $relMap = @{}
    foreach ($rel in $rels.SelectNodes("//r:Relationship", $rns)) {
        $relMap[$rel.Id] = $rel.Target
    }
    $result = @()
    foreach ($node in $nodes) {
        $rId = $node.GetAttribute("id", "http://schemas.openxmlformats.org/officeDocument/2006/relationships")
        $target = if ($relMap.ContainsKey($rId)) { $relMap[$rId] } else { $null }
        $result += [pscustomobject]@{
            Name   = $node.name
            SheetId = $node.sheetId
            Target = $target
        }
    }
    $result
}

<#
.SYNOPSIS
Parse the shared strings table from an xlsx zip.
.PARAMETER Zip
The ZipArchive object.
.EXAMPLE
$strings = _xls_shared_strings -Zip $zip
#>
function _xls_shared_strings {
    param([Parameter(Mandatory = $true)][System.IO.Compression.ZipArchive]$Zip)
    $entry = $Zip.Entries | Where-Object { $_.FullName -eq "xl/sharedStrings.xml" } | Select-Object -First 1
    if ($null -eq $entry) { return @() }
    $stream = $entry.Open()
    try {
        $reader = New-Object System.IO.StreamReader($stream)
        $text = $reader.ReadToEnd()
        $reader.Close()
    } finally {
        $stream.Close()
    }
    [xml]$doc = $text
    $ns = New-Object System.Xml.XmlNamespaceManager($doc.NameTable)
    $ns.AddNamespace("s", "http://schemas.openxmlformats.org/spreadsheetml/2006/main")
    $items = $doc.SelectNodes("//s:si", $ns)
    $strings = @()
    foreach ($si in $items) {
        $strings += $si.InnerText
    }
    $strings
}

<#
.SYNOPSIS
Parse rows from a sheet XML entry. Returns arrays of cell values.
.PARAMETER Zip
The ZipArchive object.
.PARAMETER SheetTarget
Relative path to the sheet XML inside xl/ (e.g. worksheets/sheet1.xml).
.EXAMPLE
$rows = _xls_parse_sheet -Zip $zip -SheetTarget "worksheets/sheet1.xml"
#>
function _xls_parse_sheet {
    param(
        [Parameter(Mandatory = $true)]
        [System.IO.Compression.ZipArchive]$Zip,

        [Parameter(Mandatory = $true)]
        [string]$SheetTarget
    )
    $entryPath = "xl/$SheetTarget"
    $doc = _xls_read_entry -Zip $Zip -EntryPath $entryPath
    $ns = New-Object System.Xml.XmlNamespaceManager($doc.NameTable)
    $ns.AddNamespace("s", "http://schemas.openxmlformats.org/spreadsheetml/2006/main")
    $shared = _xls_shared_strings -Zip $Zip
    $rows = $doc.SelectNodes("//s:sheetData/s:row", $ns)
    $result = @()
    foreach ($row in $rows) {
        $cells = $row.SelectNodes("s:c", $ns)
        $maxCol = 0
        $cellMap = @{}
        foreach ($c in $cells) {
            $ref = $c.r
            $colLetters = ($ref -replace '[0-9]', '')
            $colIdx = 0
            foreach ($ch in $colLetters.ToCharArray()) {
                $colIdx = $colIdx * 26 + ([int][char]$ch - [int][char]'A' + 1)
            }
            $colIdx -= 1
            if ($colIdx -gt $maxCol) { $maxCol = $colIdx }
            $val = ""
            $vNode = $c.SelectSingleNode("s:v", $ns)
            if ($null -ne $vNode) {
                $raw = $vNode.InnerText
                $cellType = $c.GetAttribute("t")
                if ($cellType -eq "s" -and $shared.Count -gt 0) {
                    $idx = [int]$raw
                    if ($idx -lt $shared.Count) { $val = $shared[$idx] }
                } else {
                    $val = $raw
                }
            }
            $cellMap[$colIdx] = $val
        }
        $rowArr = @()
        for ($i = 0; $i -le $maxCol; $i++) {
            if ($cellMap.ContainsKey($i)) { $rowArr += $cellMap[$i] }
            else { $rowArr += "" }
        }
        $result += , $rowArr
    }
    $result
}

<#
.SYNOPSIS
Resolve a sheet name to its target path inside the xlsx.
.PARAMETER Zip
The ZipArchive object.
.PARAMETER Sheet
Sheet name (defaults to the first sheet).
.EXAMPLE
$target = _xls_resolve_sheet -Zip $zip -Sheet "Sheet1"
#>
function _xls_resolve_sheet {
    param(
        [Parameter(Mandatory = $true)]
        [System.IO.Compression.ZipArchive]$Zip,

        [string]$Sheet
    )
    $sheets = _xls_sheet_list -Zip $Zip
    if ($sheets.Count -eq 0) { throw "No sheets found in workbook." }
    if ([string]::IsNullOrEmpty($Sheet)) {
        return $sheets[0]
    }
    $match = $sheets | Where-Object { $_.Name -eq $Sheet } | Select-Object -First 1
    if ($null -eq $match) {
        $names = ($sheets | ForEach-Object { $_.Name }) -join ", "
        throw "Sheet '$Sheet' not found. Available: $names"
    }
    $match
}

# =============================================================================
# Public functions
# =============================================================================

<#
.SYNOPSIS
List all sheet names in an xlsx file.
.DESCRIPTION
Opens the xlsx archive and reads sheet metadata from the workbook.
Returns one object per sheet with its name and internal ID.
.PARAMETER FilePath
Path to the xlsx file.
.EXAMPLE
xls_sheets -FilePath "C:\reports\data.xlsx"
#>
function xls_sheets {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath
    )
    _assert_path_exists -Path $FilePath
    $zip = _xls_open_zip -FilePath $FilePath
    try {
        $sheets = _xls_sheet_list -Zip $zip
        foreach ($s in $sheets) {
            [pscustomobject]@{
                Name    = $s.Name
                SheetId = $s.SheetId
            }
        }
    } finally {
        $zip.Dispose()
    }
}

<#
.SYNOPSIS
Show file metadata for an xlsx file.
.DESCRIPTION
Returns file size, sheet count, sheet names, and file timestamps.
.PARAMETER FilePath
Path to the xlsx file.
.EXAMPLE
xls_info -FilePath "C:\reports\data.xlsx"
#>
function xls_info {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath
    )
    _assert_path_exists -Path $FilePath
    $file = Get-Item -LiteralPath $FilePath
    $zip = _xls_open_zip -FilePath $FilePath
    try {
        $sheets = _xls_sheet_list -Zip $zip
        [pscustomobject]@{
            FileName     = $file.Name
            SizeKB       = [math]::Round($file.Length / 1024, 1)
            SheetCount   = $sheets.Count
            SheetNames   = ($sheets | ForEach-Object { $_.Name }) -join ", "
            Created      = $file.CreationTime.ToString("yyyy-MM-dd HH:mm:ss")
            LastModified = $file.LastWriteTime.ToString("yyyy-MM-dd HH:mm:ss")
        }
    } finally {
        $zip.Dispose()
    }
}

<#
.SYNOPSIS
Preview the first N rows of a sheet in an xlsx file.
.DESCRIPTION
Parses the sheet XML and returns structured row data.
The first row is treated as headers when possible.
.PARAMETER FilePath
Path to the xlsx file.
.PARAMETER Sheet
Sheet name to preview. Defaults to the first sheet.
.PARAMETER Rows
Number of rows to return (default 10).
.EXAMPLE
xls_preview -FilePath "C:\reports\data.xlsx"
.EXAMPLE
xls_preview -FilePath "C:\reports\data.xlsx" -Sheet "Sales" -Rows 20
#>
function xls_preview {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,

        [string]$Sheet,

        [int]$Rows = 10
    )
    _assert_path_exists -Path $FilePath
    $zip = _xls_open_zip -FilePath $FilePath
    try {
        $info = _xls_resolve_sheet -Zip $zip -Sheet $Sheet
        $allRows = _xls_parse_sheet -Zip $zip -SheetTarget $info.Target
        if ($allRows.Count -eq 0) {
            return [pscustomobject]@{ Sheet = $info.Name; Message = "Sheet is empty" }
        }
        $headers = $allRows[0]
        $dataRows = @()
        $end = [math]::Min($allRows.Count, $Rows + 1)
        for ($i = 1; $i -lt $end; $i++) {
            $obj = [ordered]@{}
            $row = $allRows[$i]
            for ($c = 0; $c -lt $headers.Count; $c++) {
                $hdr = if ([string]::IsNullOrWhiteSpace($headers[$c])) { "Col$($c+1)" } else { $headers[$c] }
                $val = if ($c -lt $row.Count) { $row[$c] } else { "" }
                $obj[$hdr] = $val
            }
            $dataRows += [pscustomobject]$obj
        }
        $dataRows
    } finally {
        $zip.Dispose()
    }
}

<#
.SYNOPSIS
Export a sheet from an xlsx file to CSV.
.DESCRIPTION
Reads the specified sheet and writes all rows as a CSV file.
Uses the first row as headers.
.PARAMETER FilePath
Path to the xlsx file.
.PARAMETER Sheet
Sheet name to export. Defaults to the first sheet.
.PARAMETER OutputPath
Path for the output CSV file. Defaults to the same directory with .csv extension.
.PARAMETER Delimiter
CSV delimiter character (default comma).
.EXAMPLE
xls_to_csv -FilePath "C:\reports\data.xlsx"
.EXAMPLE
xls_to_csv -FilePath "C:\reports\data.xlsx" -Sheet "Sales" -Delimiter ";"
#>
function xls_to_csv {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,

        [string]$Sheet,

        [string]$OutputPath,

        [string]$Delimiter = ","
    )
    _assert_path_exists -Path $FilePath
    $zip = _xls_open_zip -FilePath $FilePath
    try {
        $info = _xls_resolve_sheet -Zip $zip -Sheet $Sheet
        $allRows = _xls_parse_sheet -Zip $zip -SheetTarget $info.Target
        if ($allRows.Count -eq 0) {
            throw "Sheet '$($info.Name)' is empty, nothing to export."
        }
        if ([string]::IsNullOrEmpty($OutputPath)) {
            $base = [System.IO.Path]::ChangeExtension($FilePath, $null).TrimEnd('.')
            $safeName = $info.Name -replace '[^\w]', '_'
            $OutputPath = "${base}_${safeName}.csv"
        }
        $lines = @()
        foreach ($row in $allRows) {
            $escaped = foreach ($cell in $row) {
                $s = [string]$cell
                if ($s.Contains($Delimiter) -or $s.Contains('"') -or $s.Contains("`n")) {
                    '"' + $s.Replace('"', '""') + '"'
                } else {
                    $s
                }
            }
            $lines += ($escaped -join $Delimiter)
        }
        [System.IO.File]::WriteAllLines($OutputPath, $lines, [System.Text.Encoding]::UTF8)
        [pscustomobject]@{
            Sheet      = $info.Name
            OutputPath = $OutputPath
            RowCount   = $allRows.Count - 1
        }
    } finally {
        $zip.Dispose()
    }
}

<#
.SYNOPSIS
Search for a text value across all sheets of an xlsx file.
.DESCRIPTION
Scans every cell in every sheet for a substring match and returns
the sheet name, cell reference, and cell value for each hit.
.PARAMETER FilePath
Path to the xlsx file.
.PARAMETER Value
Text to search for (case-insensitive substring match).
.PARAMETER Limit
Maximum number of results to return (default 50).
.EXAMPLE
xls_search -FilePath "C:\reports\data.xlsx" -Value "revenue"
.EXAMPLE
xls_search -FilePath "C:\reports\data.xlsx" -Value "error" -Limit 10
#>
function xls_search {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,

        [Parameter(Mandatory = $true)]
        [string]$Value,

        [int]$Limit = 50
    )
    _assert_path_exists -Path $FilePath
    $zip = _xls_open_zip -FilePath $FilePath
    try {
        $sheets = _xls_sheet_list -Zip $zip
        $found = 0
        foreach ($s in $sheets) {
            if ($found -ge $Limit) { break }
            if ($null -eq $s.Target) { continue }
            $allRows = _xls_parse_sheet -Zip $zip -SheetTarget $s.Target
            for ($r = 0; $r -lt $allRows.Count; $r++) {
                if ($found -ge $Limit) { break }
                $row = $allRows[$r]
                for ($c = 0; $c -lt $row.Count; $c++) {
                    if ($found -ge $Limit) { break }
                    $cell = [string]$row[$c]
                    if ($cell -like "*$Value*") {
                        $colLetter = ""
                        $tmp = $c
                        do {
                            $colLetter = [char]([int][char]'A' + ($tmp % 26)) + $colLetter
                            $tmp = [math]::Floor($tmp / 26) - 1
                        } while ($tmp -ge 0)
                        [pscustomobject]@{
                            Sheet = $s.Name
                            Cell  = "$colLetter$($r + 1)"
                            Value = $cell
                        }
                        $found++
                    }
                }
            }
        }
        if ($found -eq 0) {
            [pscustomobject]@{ Message = "No matches found for '$Value'." }
        }
    } finally {
        $zip.Dispose()
    }
}
