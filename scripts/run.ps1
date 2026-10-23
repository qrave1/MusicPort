param([switch]$Build)
$taskOldCgo = $env:CGO_ENABLED
$taskOldPath = $env:PATH
$taskOldFlags = $env:GOFLAGS
$taskRoot = Split-Path $PSScriptRoot -Parent
try {
    $env:CGO_ENABLED = '1'
    if ($env:GOFLAGS -match '(^|[ ,=])ci([ ,]|$)') { throw 'Remove the ci build tag from GOFLAGS to launch the desktop GUI.' }
    $taskCompiler = Join-Path $taskRoot '.tools/mingw64/bin'
    if (Test-Path (Join-Path $taskCompiler 'gcc.exe')) { $env:PATH = "$taskCompiler;$env:PATH" }
    if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) { throw 'Fyne requires GCC/MinGW-w64 in PATH or .tools/mingw64/bin.' }
    Push-Location $taskRoot
    try {
        if ($Build) { go build -ldflags '-H=windowsgui' -o musicport.exe . } else { go run . }
        if ($LASTEXITCODE -ne 0) { throw "Go exited with code $LASTEXITCODE" }
    } finally { Pop-Location }
} finally { $env:CGO_ENABLED = $taskOldCgo; $env:PATH = $taskOldPath; $env:GOFLAGS = $taskOldFlags }
