param(
    [string]$BaseUrl = "http://localhost:8080/api/v1",
    [string]$Owner = "golang",
    [string]$Repo = "go",
    [string]$OutFile = "curl-search-response.json"
)

$ErrorActionPreference = "Stop"

function Show-Step([string]$Text) { Write-Host "`n=== $Text ===" -ForegroundColor Cyan }
function Assert-Code([int]$Expected, [string]$Actual) {
    if ([int]$Actual -ne $Expected) { throw "Expected HTTP $Expected, got $Actual" }
    Write-Host "OK: HTTP $Actual" -ForegroundColor Green
}

Show-Step "GET health: headers and body"
curl.exe --fail-with-body --silent --show-error --include `
    -H "Accept: application/json" "$BaseUrl/health"

Show-Step "GET search: query parameters, save body"
$code = curl.exe --silent --show-error --output $OutFile --write-out "%{http_code}" --get `
    -H "Accept: application/json" -H "X-Request-ID: curl-powershell-search" `
    --data-urlencode "q=http server" --data-urlencode "language=Go" `
    --data "sort=stars" --data "order=desc" --data "page=1" --data "per_page=5" `
    "$BaseUrl/repositories/search"
Assert-Code 200 $code
Write-Host "Body saved to $((Resolve-Path $OutFile).Path)"

Show-Step "Only response headers (ordinary GET, body discarded)"
curl.exe --silent --show-error --dump-header - --output NUL "$BaseUrl/health"

Show-Step "Verbose repository request"
curl.exe --verbose --output NUL -H "Accept: application/json" `
    "$BaseUrl/repositories/$Owner/$Repo"

Show-Step "POST favorite: JSON body"
curl.exe --silent --show-error --output NUL --request DELETE "$BaseUrl/favorites/$Owner/$Repo"
$favoriteFile = Join-Path $env:TEMP "reposcope-favorite.json"
$body = '{"owner":"' + $Owner + '","repo":"' + $Repo + '"}'
$code = curl.exe --silent --show-error --output $favoriteFile --write-out "%{http_code}" `
    --request POST -H "Accept: application/json" -H "Content-Type: application/json" `
    --data $body "$BaseUrl/favorites"
Assert-Code 201 $code

Show-Step "POST duplicate: expected 409 Conflict"
$errorFile = Join-Path $env:TEMP "reposcope-error.json"
$code = curl.exe --silent --show-error --output $errorFile --write-out "%{http_code}" `
    --request POST -H "Content-Type: application/json" --data $body "$BaseUrl/favorites"
Assert-Code 409 $code

Show-Step "DELETE favorite: path parameters"
$code = curl.exe --silent --show-error --output NUL --write-out "%{http_code}" `
    --request DELETE "$BaseUrl/favorites/$Owner/$Repo"
Assert-Code 204 $code

Show-Step "Negative validation scenario"
$validationFile = Join-Path $env:TEMP "reposcope-validation.json"
$code = curl.exe --silent --show-error --output $validationFile --write-out "%{http_code}" `
    "$BaseUrl/repositories/search?q=x&page=0"
Assert-Code 422 $code

Write-Host "`nAll cURL scenarios completed." -ForegroundColor Green
