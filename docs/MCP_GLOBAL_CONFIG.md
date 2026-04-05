# ✅ MCP Configurado GLOBALMENTE

## Estado Actual

El servidor MCP `prism` está ahora configurado **globalmente** y disponible desde **cualquier directorio**.

### Verificación:

```powershell
# Desde CUALQUIER directorio
claude mcp list

# Resultado:
# prism: ✓ Connected
```

## 🎯 Cómo usar

### Desde cualquier directorio:

```bash
# Ejemplo 1: Desde G1-Practica
cd C:\Users\rebel\G1-Practica
claude
"Busca funciones relacionadas con login en mi código"

# Ejemplo 2: Desde cualquier otro proyecto
cd C:\Users\rebel\MiOtroProyecto
claude
"Lista funciones disponibles"
```

### ⚠️ Nota importante:

El servidor MCP tiene acceso al código indexado en:
```
C:\Users\rebel\GolandProjects\TokenCompressorUI\code_graph.db
```

Si quieres buscar en **otro proyecto**, necesitas:

1. Indexar ese proyecto:
   ```bash
   cd C:\Users\rebel\MiOtroProyecto
   C:\Users\rebel\GolandProjects\TokenCompressorUI\prism.exe index -repo .
   ```

2. Usar la base de datos generada:
   ```bash
   # El MCP actual busca en TokenCompressorUI
   # Para otros proyectos, puedes:
   # - Crear múltiples servidores MCP (uno por proyecto)
   # - O cambiar la base de datos del servidor actual
   ```

## 📊 Herramientas disponibles globalmente

En **cualquier directorio** donde uses `claude`, tendrás acceso a:

- `search_context` - Búsqueda semántica
- `get_file_smart` - Obtener función específica
- `trace_impact` - Análisis de impacto
- `list_functions` - Listar funciones

**Pero recuerda:** Buscan en el código indexado en `code_graph.db` de TokenCompressorUI.

## 🔧 Comandos útiles

```bash
# Ver configuración del servidor
claude mcp get prism

# Ver logs
Get-Content C:\Users\rebel\GolandProjects\TokenCompressorUI\mcp_server.log -Tail 50

# Remover si lo necesitas
claude mcp remove prism -s user
```

## 💡 Configuración multi-proyecto (opcional)

Si trabajas en varios proyectos y quieres un servidor MCP por proyecto:

```bash
# Proyecto 1: TokenCompressorUI
claude mcp add -s user token-ui C:\...\prism.exe -- serve -db C:\...\TokenCompressorUI\code_graph.db

# Proyecto 2: G1-Practica
# Primero, indexa el proyecto
cd C:\Users\rebel\G1-Practica
C:\...\prism.exe index -repo .

# Luego, agrega el servidor MCP
claude mcp add -s user g1-practica C:\...\prism.exe -- serve -db C:\Users\rebel\G1-Practica\code_graph.db
```

Así tendrás múltiples servidores MCP, uno por proyecto.

## ✅ Configuración actual:

```
Scope: Global (user)
Disponible: Desde cualquier directorio
Base de datos: C:\...\TokenCompressorUI\code_graph.db
Estado: ✓ Connected
```
