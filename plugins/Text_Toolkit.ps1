# =============================================================================
# TEXT TOOLKIT – Encoding, hashing and text conversion utilities (standalone)
# Encode/decode Base64, JWT, URLs; generate UUIDs, hashes; format JSON.
# Safety: Read-only — no destructive operations.
# Entry point: txt_*
#
# FUNCTIONS
#   txt_base64_encode
#   txt_base64_decode
#   txt_jwt_decode
#   txt_url_encode
#   txt_url_decode
#   txt_uuid
#   txt_hash
#   txt_json_format
#   txt_timestamp
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
_assert_command_available -Name openssl
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

<#
.SYNOPSIS
Decode a Base64url-encoded string to plain UTF-8 text.
.PARAMETER Base64Url
Base64url-encoded input (no padding required).
.EXAMPLE
_base64url_decode -Base64Url "eyJhbGciOiJIUzI1NiJ9"
#>
function _base64url_decode {
    param([Parameter(Mandatory = $true)][string]$Base64Url)
    $s = $Base64Url.Replace('-', '+').Replace('_', '/')
    switch ($s.Length % 4) {
        2 { $s += '==' }
        3 { $s += '='  }
    }
    [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($s))
}

# -----------------------------------------------------------------------------
# Base64
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Encode a string to Base64.
.PARAMETER Text
Plain text to encode.
.EXAMPLE
txt_base64_encode -Text "hello world"
#>
function txt_base64_encode {
    param([Parameter(Mandatory = $true, Position = 0)][string]$Text)
    [System.Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($Text))
}

<#
.SYNOPSIS
Decode a Base64 string to plain text.
.PARAMETER Text
Base64-encoded string.
.EXAMPLE
txt_base64_decode -Text "aGVsbG8gd29ybGQ="
#>
function txt_base64_decode {
    param([Parameter(Mandatory = $true, Position = 0)][string]$Text)
    [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($Text))
}

# -----------------------------------------------------------------------------
# JWT
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Decode a JWT token and display header and payload as JSON.
.DESCRIPTION
Splits the JWT into its three parts and Base64url-decodes the header and
payload. Does not verify the signature.
.PARAMETER Token
JWT string (three dot-separated segments).
.EXAMPLE
txt_jwt_decode -Token "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
#>
function txt_jwt_decode {
    param([Parameter(Mandatory = $true, Position = 0)][string]$Token)

    $parts = $Token.Split('.')
    if ($parts.Count -lt 2) { throw "Invalid JWT: expected at least 2 dot-separated segments." }

    $header  = _base64url_decode -Base64Url $parts[0]
    $payload = _base64url_decode -Base64Url $parts[1]

    return [pscustomobject]@{
        Header  = $header
        Payload = $payload
    }
}

# -----------------------------------------------------------------------------
# URL encoding
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
URL-encode a string.
.PARAMETER Text
String to encode.
.EXAMPLE
txt_url_encode -Text "hello world&foo=bar"
#>
function txt_url_encode {
    param([Parameter(Mandatory = $true, Position = 0)][string]$Text)
    [System.Uri]::EscapeDataString($Text)
}

<#
.SYNOPSIS
URL-decode a string.
.PARAMETER Text
URL-encoded string.
.EXAMPLE
txt_url_decode -Text "hello%20world%26foo%3Dbar"
#>
function txt_url_decode {
    param([Parameter(Mandatory = $true, Position = 0)][string]$Text)
    [System.Uri]::UnescapeDataString($Text)
}

# -----------------------------------------------------------------------------
# UUID
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Generate a new random UUID (v4).
.EXAMPLE
txt_uuid
#>
function txt_uuid {
    [System.Guid]::NewGuid().ToString()
}

# -----------------------------------------------------------------------------
# Hashing
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Compute a hash of a string or file.
.DESCRIPTION
Computes SHA256 by default. Supports MD5, SHA1, SHA256, SHA384, SHA512.
When Path is given, hashes the file contents; otherwise hashes the Text parameter.
.PARAMETER Text
String to hash (mutually exclusive with Path).
.PARAMETER Path
File path to hash (mutually exclusive with Text).
.PARAMETER Algorithm
Hash algorithm (default: SHA256).
.EXAMPLE
txt_hash -Text "hello"
.EXAMPLE
txt_hash -Path "C:\file.txt" -Algorithm MD5
#>
function txt_hash {
    param(
        [Parameter(Position = 0)][string]$Text,
        [string]$Path,
        [ValidateSet("MD5", "SHA1", "SHA256", "SHA384", "SHA512")]
        [string]$Algorithm = "SHA256"
    )

    if (-not [string]::IsNullOrEmpty($Path)) {
        if (-not (Test-Path -LiteralPath $Path)) { throw "File not found: $Path" }
        $result = Get-FileHash -LiteralPath $Path -Algorithm $Algorithm
        return [pscustomobject]@{
            Algorithm = $result.Algorithm
            Hash      = $result.Hash
            Source    = $Path
        }
    }

    if ([string]::IsNullOrEmpty($Text)) { throw "Provide either -Text or -Path." }

    $hasher = [System.Security.Cryptography.HashAlgorithm]::Create($Algorithm)
    $bytes  = $hasher.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($Text))
    $hex    = ($bytes | ForEach-Object { $_.ToString("x2") }) -join ''

    return [pscustomobject]@{
        Algorithm = $Algorithm
        Hash      = $hex
        Source    = "(string)"
    }
}

# -----------------------------------------------------------------------------
# JSON
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Pretty-print a JSON string.
.DESCRIPTION
Parses the input JSON and re-serializes it with indentation. If no input is
given, reads from the clipboard.
.PARAMETER Json
Raw JSON string to format.
.EXAMPLE
txt_json_format -Json '{"name":"dm","version":1}'
.EXAMPLE
txt_json_format
#>
function txt_json_format {
    param([Parameter(Position = 0)][string]$Json)

    if ([string]::IsNullOrWhiteSpace($Json)) {
        $Json = Get-Clipboard
        if ([string]::IsNullOrWhiteSpace($Json)) { throw "No JSON provided and clipboard is empty." }
    }

    $Json | ConvertFrom-Json | ConvertTo-Json -Depth 20
}

# -----------------------------------------------------------------------------
# Timestamp
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Convert between Unix timestamps and human-readable dates.
.DESCRIPTION
If Value looks like a Unix epoch (all digits), converts it to a UTC datetime.
Otherwise parses it as a datetime string and returns the Unix epoch.
.PARAMETER Value
Unix timestamp (seconds or milliseconds) or a date string.
.EXAMPLE
txt_timestamp -Value "1700000000"
.EXAMPLE
txt_timestamp -Value "2023-11-14T22:13:20Z"
#>
function txt_timestamp {
    param([Parameter(Mandatory = $true, Position = 0)][string]$Value)

    $epoch = [datetime]::new(1970, 1, 1, 0, 0, 0, [System.DateTimeKind]::Utc)

    if ($Value -match '^\d+$') {
        $num = [long]$Value
        if ($num -gt 9999999999) { $num = [math]::Floor($num / 1000) }
        $dt = $epoch.AddSeconds($num)
        return [pscustomobject]@{
            Unix     = [long]($dt - $epoch).TotalSeconds
            UnixMs   = [long]($dt - $epoch).TotalMilliseconds
            DateTime = $dt.ToString("yyyy-MM-dd HH:mm:ss UTC")
        }
    }

    $dt = [datetime]::Parse($Value).ToUniversalTime()
    return [pscustomobject]@{
        Unix     = [long]($dt - $epoch).TotalSeconds
        UnixMs   = [long]($dt - $epoch).TotalMilliseconds
        DateTime = $dt.ToString("yyyy-MM-dd HH:mm:ss UTC")
    }
}
