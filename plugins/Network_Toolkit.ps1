# =============================================================================
# NETWORK TOOLKIT – HTTP and network diagnostic layer (standalone)
# HTTP requests, downloads, certificate inspection and connectivity checks.
# Safety: Read-only defaults. net_download writes files to disk.
# Entry point: net_*
#
# FUNCTIONS
#   net_http
#   net_download
#   net_ip_public
#   net_trace
#   net_cert
#   net_speed
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
_assert_command_available -Name curl
#>
function _assert_command_available {
    param([Parameter(Mandatory = $true)][string]$Name)
    if (-not (Get-Command -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Required command '$Name' was not found in PATH."
    }
}

# -----------------------------------------------------------------------------
# HTTP
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Perform an HTTP request and show status, headers and body.
.DESCRIPTION
Sends an HTTP request to the specified URL and returns status code,
headers and response body. Defaults to GET method.
.PARAMETER Url
Target URL.
.PARAMETER Method
HTTP method (default GET).
.PARAMETER Body
Optional request body for POST/PUT.
.PARAMETER ContentType
Content-Type header (default application/json).
.EXAMPLE
net_http -Url "https://httpbin.org/get"
.EXAMPLE
net_http -Url "https://httpbin.org/post" -Method POST -Body '{"key":"value"}'
#>
function net_http {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url,

        [Parameter(Mandatory = $false)]
        [ValidateSet("GET", "POST", "PUT", "DELETE", "PATCH", "HEAD")]
        [string]$Method = "GET",

        [Parameter(Mandatory = $false)]
        [string]$Body = "",

        [Parameter(Mandatory = $false)]
        [string]$ContentType = "application/json"
    )

    $params = @{
        Uri                = $Url
        Method             = $Method
        UseBasicParsing    = $true
        TimeoutSec         = 15
        ErrorAction        = "Stop"
    }

    if ($Body -and $Method -in @("POST", "PUT", "PATCH")) {
        $params["Body"]        = $Body
        $params["ContentType"] = $ContentType
    }

    try {
        $response = Invoke-WebRequest @params

        $headers = @{}
        foreach ($key in $response.Headers.Keys) {
            $headers[$key] = $response.Headers[$key] -join ", "
        }

        [pscustomobject]@{
            StatusCode  = $response.StatusCode
            StatusText  = $response.StatusDescription
            Headers     = $headers
            Body        = $response.Content.Substring(0, [Math]::Min($response.Content.Length, 4096))
        }
    }
    catch {
        $ex = $_.Exception
        if ($ex.Response) {
            [pscustomobject]@{
                StatusCode = [int]$ex.Response.StatusCode
                StatusText = $ex.Response.StatusDescription
                Error      = $ex.Message
            }
        }
        else {
            throw "HTTP request failed: $($ex.Message)"
        }
    }
}

<#
.SYNOPSIS
Download a file from a URL.
.DESCRIPTION
Downloads a file from the specified URL and saves it to the given path.
Defaults to saving in the current directory with the filename from the URL.
.PARAMETER Url
Source URL.
.PARAMETER OutFile
Output file path. Defaults to filename from URL in current directory.
.EXAMPLE
net_download -Url "https://example.com/file.zip"
.EXAMPLE
net_download -Url "https://example.com/data.json" -OutFile "C:\temp\data.json"
#>
function net_download {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url,

        [Parameter(Mandatory = $false)]
        [string]$OutFile = ""
    )

    if ([string]::IsNullOrWhiteSpace($OutFile)) {
        $uri = [System.Uri]::new($Url)
        $fileName = [System.IO.Path]::GetFileName($uri.LocalPath)
        if ([string]::IsNullOrWhiteSpace($fileName)) {
            $fileName = "download"
        }
        $OutFile = Join-Path (Get-Location).Path $fileName
    }

    Invoke-WebRequest -Uri $Url -OutFile $OutFile -UseBasicParsing -TimeoutSec 60

    $info = Get-Item -LiteralPath $OutFile
    [pscustomobject]@{
        Status = "downloaded"
        Path   = $info.FullName
        Size   = "{0:N2} KB" -f ($info.Length / 1KB)
    }
}

<#
.SYNOPSIS
Show public IP address.
.DESCRIPTION
Queries an external service to determine the public-facing IP address.
.EXAMPLE
net_ip_public
#>
function net_ip_public {
    $ip = (Invoke-RestMethod -Uri "https://api.ipify.org?format=json" -UseBasicParsing -TimeoutSec 10).ip
    [pscustomobject]@{
        PublicIP = $ip
    }
}

# -----------------------------------------------------------------------------
# Diagnostics
# -----------------------------------------------------------------------------

<#
.SYNOPSIS
Trace route to a host.
.DESCRIPTION
Runs tracert to show the network path to the specified host.
.PARAMETER Host
Target hostname or IP address.
.PARAMETER MaxHops
Maximum number of hops (default 15).
.EXAMPLE
net_trace -Host "google.com"
.EXAMPLE
net_trace -Host "8.8.8.8" -MaxHops 10
#>
function net_trace {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Host,

        [Parameter(Mandatory = $false)]
        [int]$MaxHops = 15
    )

    tracert -h $MaxHops -w 1000 $Host
}

<#
.SYNOPSIS
Show SSL certificate info for a website.
.DESCRIPTION
Connects to the specified host on port 443 and retrieves the
SSL/TLS certificate details including expiration date and issuer.
.PARAMETER Host
Hostname to inspect (without https://).
.EXAMPLE
net_cert -Host "google.com"
.EXAMPLE
net_cert -Host "github.com"
#>
function net_cert {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Host
    )

    $cleanHost = $Host -replace '^https?://', '' -replace '/.*$', ''

    try {
        $request = [System.Net.HttpWebRequest]::Create("https://$cleanHost")
        $request.Timeout = 10000
        $request.AllowAutoRedirect = $false
        $response = $request.GetResponse()
        $response.Close()
    }
    catch {
        # Connection may fail but cert is still captured
    }

    $cert = $request.ServicePoint.Certificate
    if ($null -eq $cert) {
        throw "Could not retrieve certificate for '$cleanHost'."
    }

    $cert2 = [System.Security.Cryptography.X509Certificates.X509Certificate2]::new($cert)
    $daysLeft = ($cert2.NotAfter - (Get-Date)).Days

    [pscustomobject]@{
        Host       = $cleanHost
        Subject    = $cert2.Subject
        Issuer     = $cert2.Issuer
        ValidFrom  = $cert2.NotBefore.ToString("yyyy-MM-dd")
        ValidTo    = $cert2.NotAfter.ToString("yyyy-MM-dd")
        DaysLeft   = $daysLeft
        Thumbprint = $cert2.Thumbprint
    }
}

<#
.SYNOPSIS
Quick download speed test.
.DESCRIPTION
Downloads a test file and measures the transfer speed.
Uses Cloudflare speed test endpoint.
.EXAMPLE
net_speed
#>
function net_speed {
    $url = "https://speed.cloudflare.com/__down?bytes=10000000"
    $tempFile = Join-Path $env:TEMP "dm_speedtest_$(Get-Random).bin"

    try {
        $sw = [System.Diagnostics.Stopwatch]::StartNew()
        Invoke-WebRequest -Uri $url -OutFile $tempFile -UseBasicParsing -TimeoutSec 30
        $sw.Stop()

        $fileSize = (Get-Item -LiteralPath $tempFile).Length
        $seconds = $sw.Elapsed.TotalSeconds
        $mbps = ($fileSize * 8) / ($seconds * 1000000)

        [pscustomobject]@{
            Downloaded = "{0:N2} MB" -f ($fileSize / 1MB)
            Duration   = "{0:N2} s" -f $seconds
            Speed      = "{0:N2} Mbps" -f $mbps
        }
    }
    finally {
        if (Test-Path -LiteralPath $tempFile) {
            Remove-Item -LiteralPath $tempFile -Force -ErrorAction SilentlyContinue
        }
    }
}
