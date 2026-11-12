#Requires -Version 7.2
param(
    [Parameter(Mandatory)]
    [ValidateSet('serve', 'down', 'test')]
    [string] $Action,

    [string] $MariaDBImage,

    [string] $RustFSImage
)

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
$testContainers = [System.Collections.Generic.List[string]]::new()
$devDatabase = 'traq-wslc-dev-db'
$dbPassword = 'password'

function Invoke-Wslc {
    & wslc @args

    if ($LASTEXITCODE -ne 0) {
        throw "wslc $($args[0]) failed with exit code $LASTEXITCODE."
    }
}

function Get-WslContainer([string] $Name) {
    $info = wslc inspect --type container $Name 2>$null
    if ($LASTEXITCODE -eq 0) { return $info | ConvertFrom-Json }
}

function Start-Container([string] $Name, [int] $Port, [string[]] $Options) {
    $container = Get-WslContainer $Name
    if ($null -eq $container) {
        Invoke-Wslc run --detach --rm --name $Name --publish "127.0.0.1::$Port" @Options | Out-Host
    } elseif (-not $container.State.Running) {
        Invoke-Wslc start $container.Id | Out-Host
    }

    $container = Get-WslContainer $Name
    $binding = $container.Ports."$Port/tcp" | Where-Object HostIp -eq '127.0.0.1'
    if (-not $binding.HostPort) {
        throw "Container $Name did not publish TCP port $Port on localhost."
    }

    return [int]$binding.HostPort
}

function Remove-WslContainer([string] $Name) {
    $container = Get-WslContainer $Name
    if ($null -eq $container) { return }

    if ($container.State.Running) {
        wslc stop $container.Id | Out-Host
    }

    if (Get-WslContainer $container.Id) {
        Invoke-Wslc rm --force --volumes $container.Id | Out-Host
    }
}

function Wait-Container([string] $Name, [scriptblock] $Ready) {
    $deadline = [DateTime]::UtcNow.AddSeconds(60)

    do {
        if (& $Ready) { return }

        Start-Sleep -Seconds 1
    } while ([DateTime]::UtcNow -lt $deadline)

    wslc logs $Name | Out-Host
    throw "Container $Name did not become ready."
}

if (-not (Get-Command wslc -ErrorAction SilentlyContinue)) {
    throw 'Install or update WSL to get the wslc command.'
}

if ($Action -eq 'down') {
    Remove-WslContainer $devDatabase
    exit 0
}

$isTest = $Action -eq 'test'
if (-not $MariaDBImage -or ($isTest -and -not $RustFSImage)) {
    throw 'Run the Windows tasks with mise to select the container images.'
}

Push-Location (Join-Path $PSScriptRoot '../..')

try {
    if ($isTest) {
        $testName = 'traq-wslc-test-' + [guid]::NewGuid().ToString('N')
        $dbName = "$testName-db"
        $testContainers.Add($dbName)
        $dbStorage = '--tmpfs', '/var/lib/mysql'
    } else {
        $dbName = $devDatabase
        $dbStorage = '--volume', "${devDatabase}:/var/lib/mysql"
    }

    $dbPort = Start-Container $dbName 3306 @(
        $dbStorage
        '-e', "MYSQL_ROOT_PASSWORD=$dbPassword"
        '-e', 'MYSQL_DATABASE=traq'
        $MariaDBImage
        'mysqld', '--character-set-server=utf8mb4', '--collation-server=utf8mb4_general_ci'
    )

    Wait-Container $dbName {
        wslc exec $dbName mariadb -h 127.0.0.1 -uroot "-p$dbPassword" -e 'SELECT 1' 2>$null | Out-Null
        return $LASTEXITCODE -eq 0
    }

    if ($isTest) {
        $s3Name = "$testName-s3"
        $testContainers.Add($s3Name)
        $s3Port = Start-Container $s3Name 9000 @(
            '-e', 'RUSTFS_ACCESS_KEY=GK0123456789abcdef0123456789abcdef'
            '-e', 'RUSTFS_SECRET_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef'
            '-e', 'RUSTFS_REGION=ap-northeast-1'
            '-e', 'RUSTFS_CONSOLE_ENABLE=false'
            '-e', 'RUSTFS_OBS_LOGGER_LEVEL=error'
            $RustFSImage
        )

        $s3Endpoint = "http://127.0.0.1:$s3Port"
        Wait-Container $s3Name {
            try {
                Invoke-WebRequest "$s3Endpoint/health/ready" -NoProxy -TimeoutSec 2 | Out-Null
                return $true
            } catch {
                return $false
            }
        }

        $env:MARIADB_HOSTNAME = '127.0.0.1'
        $env:MARIADB_PORT = "$dbPort"
        $env:MARIADB_USERNAME = 'root'
        $env:MARIADB_PASSWORD = $dbPassword
        $env:TRAQ_TEST_S3_ENDPOINT = $s3Endpoint
        $env:CGO_ENABLED = '1'
        go test ./... -race -shuffle=on
    } else {
        $env:TRAQ_MARIADB_HOST = '127.0.0.1'
        $env:TRAQ_MARIADB_PORT = "$dbPort"
        $env:TRAQ_MARIADB_USERNAME = 'root'
        $env:TRAQ_MARIADB_PASSWORD = $dbPassword
        $env:TRAQ_MARIADB_DATABASE = 'traq'
        $env:TRAQ_STORAGE_TYPE = 'local'
        $env:TRAQ_STORAGE_LOCAL_DIR = Join-Path (Get-Location) 'dev/data/windows/storage'
        $env:CGO_ENABLED = '0'
        New-Item -ItemType Directory -Path $env:TRAQ_STORAGE_LOCAL_DIR -Force | Out-Null

        go run . serve --dev --skip-init-emojis
    }

    $exitCode = $LASTEXITCODE
} finally {
    foreach ($name in $testContainers) {
        try {
            Remove-WslContainer $name
        } catch {
            Write-Warning "Could not remove container ${name}: $_"
            $exitCode = 1
        }
    }

    Pop-Location
}

exit $exitCode
