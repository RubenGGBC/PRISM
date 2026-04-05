# 🔍 Verificación del MCP Server

## Cómo verificar que funciona correctamente

### 1. **Revisar el log del servidor**

El MCP server ahora genera un log en:
```
C:\Users\rebel\GolandProjects\TokenCompressorUI\mcp_server.log
```

**Cuando Claude Code usa una herramienta, verás logs como:**

```log
[MCP] 2026/04/05 01:47:00 🚀 MCP Server starting...
[MCP] 2026/04/05 01:47:00 ✅ Tools registered: search_context, get_file_smart, trace_impact, list_functions
[MCP] 2026/04/05 01:47:00 📡 Starting stdio transport...
[MCP] 2026/04/05 01:47:15 🔍 search_context called: query='login functions', limit=5
[MCP] 2026/04/05 01:47:15 ✅ search_context: found 3 results (keyword) in 12ms
[MCP] 2026/04/05 01:47:23 📄 get_file_smart called: file='auth/login.ts', symbol='login'
[MCP] 2026/04/05 01:47:23 ✅ get_file_smart: found symbol with 0 callers, 4 callees in 8ms
```

### 2. **Verificar en Claude Desktop**

Después de reiniciar Claude Desktop:

1. **Abre Claude Code**
2. **Busca el indicador de MCP** (icono o mención en la UI)
3. **Pregunta algo como:**
   ```
   "Busca funciones de login usando search_context"
   ```
   o
   ```
   "Dame la función login del archivo auth/login.ts usando get_file_smart"
   ```

Si funciona, Claude responderá con los datos del código indexado.

### 3. **Ahorro de tokens - Antes vs Después**

#### **❌ SIN MCP (método tradicional):**
```
Usuario: "Explícame cómo funciona el login"

Claude debe:
1. Leer TODO el archivo auth/login.ts (1000+ tokens)
2. Leer TODO db/user.ts (500+ tokens)
3. Leer TODO utils/helpers.ts (400+ tokens)

Total: ~1900 tokens enviados
```

#### **✅ CON MCP (método inteligente):**
```
Usuario: "Explícame cómo funciona el login"

Claude usa:
1. search_context("login") → Encuentra auth/login.ts:login
2. get_file_smart(file="auth/login.ts", symbol="login")
   → Devuelve SOLO la función login + metadata (200 tokens)

Total: ~200 tokens enviados
Ahorro: 90% de tokens ⚡
```

### 4. **Test manual rápido**

```powershell
# En otra terminal
cd C:\Users\rebel\GolandProjects\TokenCompressorUI

# Asegúrate de tener datos indexados
.\prism.exe index -repo .\test\sample-repo

# Ver las estadísticas
.\prism.exe stats  # (si existe el comando)
```

Luego en Claude Code, pregunta:
```
"Lista todas las funciones en auth/login.ts usando list_functions"
```

### 5. **Monitoreo en tiempo real**

```powershell
# Abre una terminal y observa el log en tiempo real
Get-Content C:\Users\rebel\GolandProjects\TokenCompressorUI\mcp_server.log -Wait -Tail 20
```

Cada vez que Claude use una herramienta, verás el log actualizado instantáneamente.

## 📊 Métricas de ahorro

El log muestra:
- **Herramienta usada** (🔍 search_context, 📄 get_file_smart, etc.)
- **Parámetros** (query, file, symbol)
- **Resultados** (cuántos nodos encontrados)
- **Tiempo** (en milisegundos)

**Ejemplo de ahorro real:**
- Archivo completo: 2000 tokens
- get_file_smart: 150 tokens
- **Ahorro: 92.5%** 🎉

## ⚠️ Si no aparece en Claude Code

1. Verifica que el archivo de configuración existe:
   ```
   C:\Users\rebel\AppData\Roaming\Claude\claude_desktop_config.json
   ```

2. Reinicia completamente Claude Desktop (no solo la ventana)

3. Revisa que `prism.exe` existe:
   ```powershell
   Test-Path C:\Users\rebel\GolandProjects\TokenCompressorUI\prism.exe
   ```

4. Prueba ejecutar manualmente el servidor:
   ```powershell
   .\prism.exe serve -db code_graph.db
   ```
   Debería quedarse esperando conexiones y crear `mcp_server.log`
