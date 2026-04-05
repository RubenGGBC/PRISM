
#!/usr/bin/env pwsh
# Test script para verificar el MCP server

Write-Host "🧪 Testing PRISM MCP Server" -ForegroundColor Cyan
Write-Host ""

# 1. Verificar que el ejecutable existe
Write-Host "1️⃣  Verificando ejecutable..." -ForegroundColor Yellow
$exePath = "C:\Users\rebel\GolandProjects\TokenCompressorUI\prism.exe"
if (Test-Path $exePath) {
    Write-Host "   ✅ prism.exe encontrado" -ForegroundColor Green
} else {
    Write-Host "   ❌ prism.exe NO encontrado" -ForegroundColor Red
    exit 1
}

# 2. Verificar database
Write-Host ""
Write-Host "2️⃣  Verificando base de datos..." -ForegroundColor Yellow
$dbPath = "C:\Users\rebel\GolandProjects\TokenCompressorUI\code_graph.db"
if (Test-Path $dbPath) {
    $dbSize = (Get-Item $dbPath).Length
    Write-Host "   ✅ code_graph.db encontrado ($([math]::Round($dbSize/1KB, 2)) KB)" -ForegroundColor Green
} else {
    Write-Host "   ⚠️  code_graph.db NO encontrado - ejecuta: .\prism.exe index -repo .\test\sample-repo" -ForegroundColor Yellow
}

# 3. Verificar configuración de Claude
Write-Host ""
Write-Host "3️⃣  Verificando configuración de Claude..." -ForegroundColor Yellow
$claudeConfigPath = "$env:APPDATA\Claude\claude_desktop_config.json"
if (Test-Path $claudeConfigPath) {
    Write-Host "   ✅ claude_desktop_config.json encontrado" -ForegroundColor Green
    $config = Get-Content $claudeConfigPath | ConvertFrom-Json
    if ($config.mcpServers."prism") {
        Write-Host "   ✅ Servidor 'prism' configurado" -ForegroundColor Green
    } else {
        Write-Host "   ❌ Servidor 'prism' NO configurado" -ForegroundColor Red
    }
} else {
    Write-Host "   ❌ claude_desktop_config.json NO encontrado" -ForegroundColor Red
}

# 4. Verificar sample data
Write-Host ""
Write-Host "4️⃣  Verificando datos de prueba..." -ForegroundColor Yellow
$sampleFiles = @(
    "C:\Users\rebel\GolandProjects\TokenCompressorUI\test\sample-repo\auth\login.ts",
    "C:\Users\rebel\GolandProjects\TokenCompressorUI\test\sample-repo\db\user.ts",
    "C:\Users\rebel\GolandProjects\TokenCompressorUI\test\sample-repo\utils\helpers.ts"
)
$foundFiles = 0
foreach ($file in $sampleFiles) {
    if (Test-Path $file) {
        $foundFiles++
    }
}
Write-Host "   ✅ $foundFiles/3 archivos de prueba encontrados" -ForegroundColor Green

# 5. Test de comandos básicos
Write-Host ""
Write-Host "5️⃣  Probando comandos básicos..." -ForegroundColor Yellow

Push-Location C:\Users\rebel\GolandProjects\TokenCompressorUI

# Test parse
Write-Host "   🔍 Probando: prism parse..." -NoNewline
$parseOutput = & .\prism.exe parse .\test\sample-repo\auth\login.ts 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host " ✅" -ForegroundColor Green
} else {
    Write-Host " ❌" -ForegroundColor Red
}

Pop-Location

# 6. Resumen
Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "📊 RESUMEN" -ForegroundColor Cyan
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host ""
Write-Host "Para usar con Claude Code:" -ForegroundColor White
Write-Host "1. Reinicia Claude Desktop" -ForegroundColor Gray
Write-Host "2. Pregunta: 'Lista funciones en auth/login.ts usando list_functions'" -ForegroundColor Gray
Write-Host "3. Revisa el log: .\mcp_server.log" -ForegroundColor Gray
Write-Host ""
Write-Host "Monitoreo en tiempo real:" -ForegroundColor White
Write-Host "   Get-Content .\mcp_server.log -Wait -Tail 20" -ForegroundColor Gray
Write-Host ""
