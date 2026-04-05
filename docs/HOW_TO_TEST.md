# 🧪 Cómo Probar el MCP con GitHub Copilot

## Estado Actual

✅ `.mcp.json` configurado en el proyecto
✅ Servidor MCP listo para GitHub Copilot

## 📋 Opciones para Probar

### Opción 1: Probar con Claude Code CLI (Ya funciona)

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
claude

# Pregunta algo como:
"Lista todas las funciones en auth/login.ts"
```

**Esto YA funciona** y puedes ver el MCP en acción en `mcp_server.log`

### Opción 2: GitHub Copilot en VS Code (Si lo tienes instalado)

1. Abre VS Code en este proyecto:
   ```bash
   cd C:\Users\rebel\GolandProjects\TokenCompressorUI
   code .
   ```

2. Asegúrate de tener la extensión GitHub Copilot instalada

3. El `.mcp.json` será detectado automáticamente (en versiones recientes)

4. Pregunta en el chat de Copilot sobre el código

### Opción 3: GitHub Copilot Workspace (En GitHub.com)

1. Abre este repositorio en GitHub.com
2. Haz push del `.mcp.json`:
   ```bash
   git add .mcp.json
   git commit -m "Add MCP configuration"
   git push
   ```

3. Usa GitHub Copilot Workspace en el repositorio
4. El `.mcp.json` será detectado automáticamente

### Opción 4: gh copilot extension (Si quieres instalarlo)

```bash
# Instalar la extensión de GitHub Copilot para gh CLI
gh extension install github/gh-copilot

# Usar:
gh copilot suggest "lista funciones en auth/login.ts"
```

## 🔍 Cómo Verificar que Funciona

### Ver logs en tiempo real:

```powershell
# Terminal 1: Monitorear logs
Get-Content C:\Users\rebel\GolandProjects\TokenCompressorUI\mcp_server.log -Wait -Tail 20
```

Cuando GitHub Copilot use el MCP, verás:
```log
[MCP] 2026/04/05 02:00:00 🚀 MCP Server starting...
[MCP] 2026/04/05 02:00:15 🔍 search_context called: query='login'
[MCP] 2026/04/05 02:00:15 ✅ search_context: found 3 results in 12ms
```

## ⚡ Prueba AHORA con Claude Code

Ya que Claude Code está funcionando, prueba ahora:

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI

# En una terminal, monitorea logs:
Get-Content .\mcp_server.log -Wait -Tail 20

# En otra terminal, usa Claude:
claude
```

**Pregunta algo como:**
```
"Busca todas las funciones relacionadas con autenticación"
"Dame la función login de auth/login.ts"
"¿Qué funciones llaman a validateEmail?"
```

Verás en el log cómo Claude usa las herramientas MCP.

## 📊 Comparación de Tokens

### Sin MCP:
```
Tu pregunta: "Explica la función login"
Claude lee: auth/login.ts (completo)
Tokens usados: ~2000
```

### Con MCP:
```
Tu pregunta: "Explica la función login"
Claude usa: get_file_smart("auth/login.ts", "login")
Tokens usados: ~200

AHORRO: 90% 🎉
```

## 🎯 Recomendación

**Empieza probando con Claude Code CLI** (que ya funciona):

```bash
cd C:\Users\rebel\GolandProjects\TokenCompressorUI
claude
```

Luego puedes probar GitHub Copilot cuando:
- Lo tengas en VS Code
- Uses GitHub Copilot Workspace
- Instales la extensión gh copilot

El archivo `.mcp.json` estará listo para cuando lo uses.
