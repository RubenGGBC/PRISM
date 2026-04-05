# 📍 Dónde aparece el MCP en Claude Desktop

## Ubicaciones donde verás el MCP Server

### 1. **En la configuración de Claude Desktop**

**Ruta:** Settings → Developer → Model Context Protocol

Deberías ver algo como:

```
MCP Servers:
✅ prism (Connected)
   Status: Active
   Tools: 4
```

### 2. **Icono de herramientas en el chat**

Cuando Claude usa una herramienta MCP, verás:

```
🔧 Using tool: search_context
   query: "login functions"
   
[Resultado de la herramienta...]
```

### 3. **En la lista de capacidades**

En algunas versiones de Claude Desktop, hay un panel lateral que muestra:

```
Available Tools:
  📦 prism
    └─ search_context
    └─ get_file_smart
    └─ trace_impact
    └─ list_functions
```

### 4. **NO aparece como botón/menú separado**

❌ **NO busques:**
- Un botón "MCP" en la UI
- Un menú desplegable de MCP
- Una pestaña especial

✅ **Lo verás cuando:**
- Claude decida usar las herramientas automáticamente
- Mencionas explícitamente usar una herramienta
- Preguntas sobre código que está indexado

## 🧪 Prueba rápida para verificar

### En Claude Desktop, escribe:

```
Por favor, lista todas las funciones en el archivo auth/login.ts usando la herramienta list_functions
```

**Si funciona, verás:**
1. Claude invoca `list_functions`
2. Muestra los resultados del MCP
3. En `mcp_server.log` aparece el registro

### Ejemplo de respuesta esperada:

```
🔧 Calling tool: list_functions
Parameters: { "file": "auth/login.ts" }

## Found 4 results

### 1. login
**Type:** function | **File:** auth/login.ts:10

### 2. logout
**Type:** function | **File:** auth/login.ts:28

...
```

## 🔍 Si NO aparece nada:

### Verificaciones:

1. **¿Reiniciaste Claude Desktop completamente?**
   ```powershell
   # Asegúrate de cerrar TODO
   Get-Process | Where-Object {$_.Name -like "*claude*"} | Stop-Process -Force
   # Luego abre Claude Desktop de nuevo
   ```

2. **¿La configuración está en el lugar correcto?**
   ```powershell
   Get-Content "$env:APPDATA\Claude\claude_desktop_config.json"
   ```
   
   Debe mostrar:
   ```json
   {
     "mcpServers": {
       "prism": {
         "command": "C:\\Users\\rebel\\GolandProjects\\TokenCompressorUI\\prism.exe",
         "args": ["serve", "-db", "C:\\Users\\rebel\\GolandProjects\\TokenCompressorUI\\code_graph.db"]
       }
     }
   }
   ```

3. **¿El servidor se está ejecutando?**
   ```powershell
   # Prueba ejecutarlo manualmente
   cd C:\Users\rebel\GolandProjects\TokenCompressorUI
   .\prism.exe serve -db code_graph.db
   
   # Deberías ver:
   # (el proceso se queda esperando - esto es correcto)
   ```

4. **¿Hay errores en el log?**
   ```powershell
   Get-Content .\mcp_server.log
   ```

## 📋 Versiones de Claude Desktop

Dependiendo de tu versión de Claude Desktop, la UI puede variar:

### **Versión con panel de herramientas:**
- Panel lateral mostrando MCP servers
- Indicador visual cuando se usan herramientas

### **Versión sin panel dedicado:**
- Las herramientas se usan automáticamente
- Se muestran en el flujo del chat
- No hay UI separada para MCP

## 💡 Tip: Forzar el uso de MCP

Si no estás seguro si está funcionando, **menciona explícitamente la herramienta**:

```
"Usa get_file_smart para obtener la función login del archivo auth/login.ts"
```

o

```
"Busca funciones relacionadas con 'password' usando search_context"
```

De esta forma Claude **debe** usar el MCP server, y verás la evidencia en:
1. La respuesta de Claude (mostrará que usó la herramienta)
2. El archivo `mcp_server.log` (registro de la llamada)

## 🎯 Alternativa: Ver Claude Desktop Logs

Claude Desktop también tiene sus propios logs:

```powershell
# Ubicación típica (puede variar):
Get-ChildItem "$env:APPDATA\Claude\logs" -Recurse | Sort-Object LastWriteTime -Descending | Select-Object -First 5
```

Busca líneas que mencionen "MCP" o "prism".
