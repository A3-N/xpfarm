# XPFarm - Unified CLI
# Usage: .\xpfarm.ps1 [build|up|debug|onlyGo|down|help]

param(
    [Parameter(Position = 0)]
    [string]$Command = "help",

    [Parameter(Position = 1, ValueFromRemainingArguments)]
    [string[]]$ExtraArgs
)

$ErrorActionPreference = "Stop"

function Show-Banner {
    Write-Host "`e[38;2;139;92;246m ____  ________________________                     `e[0m"
    Write-Host "`e[38;2;122;98;230m `u{2572}   `u{2572}`u{2571}  `u{2571}`u{2572}______   `u{2572}_   _____`u{2571}____ _______  _____  `e[0m"
    Write-Host "`e[38;2;105;110;214m  `u{2572}     `u{2571}  `u{2502}     ___`u{2571}`u{2502}    __) `u{2572}__  `u{2572}`u{2572}_  __ `u{2572}`u{2571}     `u{2572} `e[0m"
    Write-Host "`e[38;2;80;130;190m  `u{2571}     `u{2572}  `u{2502}    `u{2502}    `u{2502}     `u{2572}   `u{2571} __ `u{2572}`u{2502}  `u{2502} `u{2572}`u{2571}  y y  `u{2572}`e[0m"
    Write-Host "`e[38;2;48;158;163m `u{2571}___`u{2571}`u{2572}  `u{2572} `u{2502}____`u{2502}    `u{2572}___  `u{2571}  (____  `u{2571}__`u{2502}  `u{2502}__`u{2502}_`u{2502}  `u{2571}`e[0m"
    Write-Host "`e[38;2;16;185;129m       `u{2572}_`u{2571}               `u{2572}`u{2571}        `u{2572}`u{2571}            `u{2572}`u{2571} `e[0m"
    Write-Host "`e[38;2;16;185;129m                                    github.com/A3-N`e[0m"
}

function Assert-Docker {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        Write-Host "`e[1;31mError: Docker is not installed`e[0m"
        exit 1
    }
}


function Invoke-Build {
    Assert-Docker
    Show-Banner
    Write-Host "`e[1mBuilding XPFarm + Overlord containers...`e[0m"
    docker compose build
    Write-Host ""
    Write-Host "`e[1;32mBuild complete!`e[0m Run `e[1m.\xpfarm.ps1 up`e[0m to start."
}

function Invoke-Up {
    Assert-Docker
    Show-Banner

    # Ensure data directory exists
    if (-not (Test-Path "data")) {
        New-Item -ItemType Directory -Force -Path "data" | Out-Null
    }

    Write-Host "`e[1mStarting XPFarm + Overlord...`e[0m"
    docker compose up -d

    Write-Host "`e[1mWaiting for XPFarm web UI...`e[0m"
    while ($true) {
        try {
            $response = Invoke-WebRequest -Uri "http://localhost:8888" -UseBasicParsing -ErrorAction Stop
            break
        } catch {
            Start-Sleep -Seconds 2
        }
    }

    Write-Host ""
    Write-Host "`e[1;32mEnvironment is running and web UI is ready!`e[0m"
    Write-Host "  XPFarm:   `e[1mhttp://localhost:8888`e[0m"
    Write-Host "  Overlord: `e[1mRunning (internal)`e[0m"
    Write-Host ""
    docker compose ps
}

function Invoke-OnlyGo {
    param([string[]]$GoArgs = @())
    Show-Banner
    Write-Host "`e[1mBuilding XPFarm (Go native, no Docker)...`e[0m"
    Write-Host "`e[1mNote: Overlord features require Docker.`e[0m"
    Write-Host ""

    go build -o xpfarm.exe main.go
    Write-Host "`e[1;32mBuild complete. Starting...`e[0m"
    if ($GoArgs.Count -gt 0) {
        & .\xpfarm.exe @GoArgs
    } else {
        & .\xpfarm.exe
    }
}

function Invoke-Debug {
    Assert-Docker
    Show-Banner

    # Ensure data directory exists
    if (-not (Test-Path "data")) {
        New-Item -ItemType Directory -Force -Path "data" | Out-Null
    }

    Write-Host "`e[1;33mStarting XPFarm + Overlord in DEBUG mode...`e[0m"
    docker compose up -d overlord
    docker compose run --rm -p 8888:8888 xpfarm ./xpfarm -debug
}

function Invoke-Down {
    Assert-Docker
    Write-Host "`e[1mStopping all containers...`e[0m"
    docker compose down
    Write-Host "`e[1;32mEnvironment stopped.`e[0m"
}

function Show-Help {
    Show-Banner
    Write-Host "Usage: " -NoNewline
    Write-Host ".\xpfarm.ps1" -NoNewline -ForegroundColor White
    Write-Host " <command>"
    Write-Host ""
    Write-Host "Commands:"
    Write-Host "  build" -NoNewline -ForegroundColor White; Write-Host "       Build the Docker containers (XPFarm + Overlord)"
    Write-Host "  up" -NoNewline -ForegroundColor White; Write-Host "          Start the environment (docker compose up)"
    Write-Host "  debug" -NoNewline -ForegroundColor White; Write-Host "       Start in debug mode (verbose logging + Gin debug)"
    Write-Host "  onlyGo" -NoNewline -ForegroundColor White; Write-Host "      Compile and run Go binary directly (no Docker, no Overlord)"
    Write-Host "  down" -NoNewline -ForegroundColor White; Write-Host "        Stop all Docker containers"
    Write-Host "  help" -NoNewline -ForegroundColor White; Write-Host "        Show this help message"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\xpfarm.ps1 build        # Build containers"
    Write-Host "  .\xpfarm.ps1 up           # Start full stack"
    Write-Host "  .\xpfarm.ps1 debug        # Start with debug logging"
    Write-Host "  .\xpfarm.ps1 onlyGo       # Dev mode, Go only"
}

switch ($Command) {
    "build"    { Invoke-Build }
    "up"       { Invoke-Up }
    "debug"    { Invoke-Debug }
    "onlyGo"   { Invoke-OnlyGo -GoArgs $ExtraArgs }
    "down"     { Invoke-Down }
    default    { Show-Help }
}
