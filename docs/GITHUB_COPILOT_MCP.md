# GitHub Copilot + MCP Integration

## ✅ Configuración para GitHub Copilot

Se ha creado el archivo `.mcp.json` en la raíz del proyecto que GitHub Copilot puede usar para acceder al servidor MCP.

### Archivo: `.mcp.json`
```json
{
  "mcpServers": {
    "prism": {
      "command": "C:\\Users\\rebel\\GolandProjects\\TokenCompressorUI\\prism.exe",
      "args": ["serve", "-db", "C:\\Users\\rebel\\GolandProjects\\TokenCompressorUI\\code_graph.db"],
      "type": "stdio"
    }
  }
}
```

## 🔧 Cómo funciona

### Para GitHub Copilot Workspace/Agents

Cuando uses GitHub Copilot en este proyecto, el archivo `.mcp.json` permite que:

1. **GitHub Copilot detecte automáticamente** el servidor MCP
2. **Acceda a las herramientas** de análisis de código
3. **Use contexto inteligente** en lugar de leer archivos completos

### Para GitHub Copilot CLI (gh co)

Si usas `gh co` (GitHub Copilot CLI), también puede utilizar MCP servers configurados localmente.

## 🧪 Probar con GitHub Copilot

### Opción 1: En GitHub.com (Copilot Workspace)

1. Abre este repositorio en GitHub
2. Usa GitHub Copilot Workspace
3. El `.mcp.json` será detectado automáticamente
4. Las herramientas MCP estarán disponibles

### Opción 2: Con gh agent-task

```bash
# Crear una tarea de agente
gh agent-task create

# El agente puede acceder a las herramientas MCP definidas en .mcp.json
```

### Opción 3: VS Code + GitHub Copilot

Si usas VS Code con GitHub Copilot:
1. Abre este proyecto
2. El `.mcp.json` puede ser detectado (dependiendo de la versión)
3. Las herramientas estarán disponibles

## 📋 Herramientas disponibles vía MCP

- `search_context` - Búsqueda semántica de código
- `get_file_smart` - Obtener función específica
- `trace_impact` - Análisis de impacto de cambios
- `list_functions` - Listar funciones

## 🔍 Verificar que funciona

### Monitorear uso:
```powershell
# Ver logs en tiempo real
Get-Content C:\Users\rebel\GolandProjects\TokenCompressorUI\mcp_server.log -Wait -Tail 20
```

Cuando GitHub Copilot use las herramientas MCP, verás logs como:
```log
[MCP] 2026/04/05 01:55:00 🔍 search_context called: query='login'
[MCP] 2026/04/05 01:55:00 ✅ search_context: found 3 results in 12ms
```

## 📊 Ahorro de tokens

**Sin MCP:**
- GitHub Copilot lee archivos completos: ~2000 tokens

**Con MCP:**
- Usa `get_file_smart`: ~200 tokens
- **Ahorro: 90%** 🎉

## ⚙️ Configuración alternativa global

Si quieres configurar MCP globalmente para GitHub Copilot:

```bash
# Crear configuración global
mkdir -p ~/.github-copilot
cat > ~/.github-copilot/mcp-servers.json << EOF
{
  "prism": {
    "command": "C:\\Users\\rebel\\GolandProjects\\TokenCompressorUI\\prism.exe",
    "args": ["serve", "-db", "code_graph.db"],
    "type": "stdio"
  }
}
EOF
```

**Nota:** La ubicación exacta puede variar según la versión de GitHub Copilot que uses.

## 🚀 Listo para usar

El servidor MCP está ahora configurado para:
- ✅ Claude Code CLI (vía `claude mcp`)
- ✅ GitHub Copilot (vía `.mcp.json`)

Ambos pueden acceder a las mismas herramientas de análisis de código inteligente.
