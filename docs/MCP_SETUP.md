# MCP Integration with Claude Code

## ✅ Configuración Completada

El servidor MCP de PRISM Platform ha sido integrado con Claude Code.

**Ubicación del archivo de configuración:**
```
C:\Users\rebel\AppData\Roaming\Claude\claude_desktop_config.json
```

**Configuración aplicada:**
```json
{
  "mcpServers": {
    "prism": {
      "command": "C:\\Users\\rebel\\GolandProjects\\TokenCompressorUI\\prism.exe",
      "args": [
        "serve",
        "-db",
        "C:\\Users\\rebel\\GolandProjects\\TokenCompressorUI\\code_graph.db"
      ]
    }
  }
}
```

## 🚀 Próximos pasos

1. **Reinicia Claude Desktop** para que cargue la nueva configuración

2. **Indexa tu repositorio** (si aún no lo hiciste):
   ```bash
   cd C:\Users\rebel\GolandProjects\TokenCompressorUI
   .\prism.exe index -repo .\tu-proyecto
   ```

3. **Genera embeddings** (opcional, para búsqueda semántica):
   ```bash
   # Requiere Ollama corriendo localmente
   .\prism.exe embed -db code_graph.db
   ```

4. **Verifica en Claude Code** que aparezca el servidor MCP en la configuración

## 🛠️ Herramientas disponibles en Claude Code

Una vez reiniciado, Claude Code tendrá acceso a:

- **`search_context`** - Búsqueda semántica de código
  ```
  Busca "funciones de autenticación"
  ```

- **`get_file_smart`** - Obtener función específica sin cargar todo el archivo
  ```
  Dame la función login del archivo auth/login.ts
  ```

- **`trace_impact`** - Analizar impacto de cambios
  ```
  ¿Qué funciones se verían afectadas si modifico validateEmail?
  ```

- **`list_functions`** - Listar funciones
  ```
  Lista todas las funciones en db/user.ts
  ```

## 🔍 Verificar que funciona

Después de reiniciar Claude Desktop, puedes pedirle a Claude:
```
"Usa search_context para buscar funciones de login"
```

Si todo está bien configurado, Claude usará el MCP server para buscar en tu código indexado.
