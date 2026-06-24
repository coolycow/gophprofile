# Сборка образов в Docker Rancher Desktop (тот же daemon, что использует k3s).
# Запускать из корня репозитория.
# Нужен, если `docker build` идёт в Docker Desktop, а Kubernetes — в Rancher Desktop.

$ErrorActionPreference = "Stop"
$Rdctl = "C:\Program Files\Rancher Desktop\resources\resources\win32\bin\rdctl.exe"
$Repo = (Get-Location).Path -replace '\\', '/' -replace '^([A-Z]):', { "/mnt/$($_.Groups[1].Value.ToLower())" }

if (-not (Test-Path $Rdctl)) {
    Write-Error "rdctl not found. Install Rancher Desktop."
}

function Build-RD {
    param([string]$Dockerfile, [string]$Tag)
    Write-Host "Building $Tag via Rancher Desktop ..."
    & $Rdctl shell -- docker build -f "/mnt/i/practicum/gophprofile/$Dockerfile" -t $Tag /mnt/i/practicum/gophprofile
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

# Путь I: → /mnt/i/ (подставьте свой диск при необходимости)
$Root = "/mnt/i/practicum/gophprofile"
& $Rdctl shell -- docker build -f "$Root/docker/server/Dockerfile" -t gophprofile/server:latest $Root
& $Rdctl shell -- docker build -f "$Root/docker/worker/Dockerfile" -t gophprofile/worker:latest $Root
& $Rdctl shell -- docker build -f "$Root/docker/client/Dockerfile" -t gophprofile/client:latest $Root

Write-Host "Verify in Rancher docker:"
& $Rdctl shell -- docker images gophprofile/server gophprofile/worker gophprofile/client
