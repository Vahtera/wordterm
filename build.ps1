# Builds standalone Wordterm executables into dist\ (PowerShell; works on Windows, Linux, macOS).
# Set $env:VERSION to stamp the About dialog.
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot
$version = if ($env:VERSION) { $env:VERSION } else { "dev" }

go vet ./...;  if ($LASTEXITCODE -ne 0) { throw "go vet failed" }
go test ./...; if ($LASTEXITCODE -ne 0) { throw "go test failed" }

New-Item -ItemType Directory -Force dist | Out-Null
$env:CGO_ENABLED = "0"
foreach ($t in "linux/amd64","linux/arm64","windows/amd64","windows/arm64","darwin/amd64","darwin/arm64") {
    $os, $arch = $t.Split("/")
    $ext = if ($os -eq "windows") { ".exe" } else { "" }
    $env:GOOS = $os; $env:GOARCH = $arch
    Write-Host "== building wordterm-$os-$arch$ext"
    go build -trimpath -ldflags "-s -w -X github.com/Vahtera/wordterm/internal/app.Version=$version" -o "dist/wordterm-$os-$arch$ext" .
    if ($LASTEXITCODE -ne 0) { throw "build failed for $t" }
}
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED
