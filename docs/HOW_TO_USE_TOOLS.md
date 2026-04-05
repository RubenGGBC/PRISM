# 🔧 Cómo Hacer que Claude Use las Herramientas MCP

## ⚠️ Problema Común

Claude no usa las herramientas MCP automáticamente a menos que:
1. Le indiques explícitamente que use una herramienta
2. La pregunta sea muy clara sobre qué archivo/función necesitas

## ✅ Forma Correcta de Preguntar

### ❌ MAL (demasiado vago):
```
"Lista funciones"
```

### ✅ BIEN (explícito):
```
"Usa la herramienta list_functions para listar todas las funciones en auth/login.ts"
```

o

```
"Con la herramienta get_file_smart, dame la función login del archivo auth/login.ts"
```

## 🎯 Ejemplos que Funcionan

### Ejemplo 1: Listar funciones
```
Usa list_functions para mostrar todas las funciones en el archivo auth/login.ts
```

**Deberías ver en el log:**
```log
[MCP] 📋 list_functions called: file='auth/login.ts'
[MCP] ✅ list_functions: found X functions in XXms
```

### Ejemplo 2: Buscar funciones
```
Usa search_context para buscar funciones relacionadas con "password"
```

**Deberías ver en el log:**
```log
[MCP] 🔍 search_context called: query='password'
[MCP] ✅ search_context: found X results in XXms
```

### Ejemplo 3: Obtener función específica
```
Usa get_file_smart para obtener la función login del archivo auth/login.ts
```

**Deberías ver en el log:**
```log
[MCP] 📄 get_file_smart called: file='auth/login.ts', symbol='login'
[MCP] ✅ get_file_smart: found symbol with X callers, X callees in XXms
```

### Ejemplo 4: Análisis de impacto
```
Usa trace_impact para analizar qué funciones se verían afectadas si modifico utils/helpers.ts:validateEmail
```

**Deberías ver en el log:**
```log
[MCP] 🎯 trace_impact called: function_id='utils/helpers.ts:validateEmail'
[MCP] ✅ trace_impact: found X direct callers, X total affected in XXms
```

## 🧪 Prueba Ahora

### En la Terminal 2 con Claude, escribe exactamente:

```
Usa la herramienta list_functions para listar todas las funciones en el archivo auth/login.ts
```

### Deberías ver:

**Terminal 1 (log):**
```log
[MCP] 📋 list_functions called: file='auth/login.ts', pattern=''
[MCP] ✅ list_functions: found 4 functions in Xms
```

**Terminal 2 (Claude):**
Claude mostrará algo como:
```
## Found 4 results

### 1. login
**Type:** function | **File:** auth/login.ts:10

### 2. logout
**Type:** function | **File:** auth/login.ts:28

### 3. createSession
**Type:** function | **File:** auth/login.ts:36

### 4. deleteSession
**Type:** function | **File:** auth/login.ts:43
```

## 💡 Tips

- **Sé explícito**: Menciona el nombre de la herramienta
- **Da contexto**: Especifica el archivo exacto
- **Verifica el log**: Siempre mira la Terminal 1 para confirmar que la herramienta se llamó

## 🎯 Comandos de Prueba

Copia y pega estos en Claude:

```
Usa list_functions para listar funciones en auth/login.ts
```

```
Usa search_context para buscar "password"
```

```
Usa get_file_smart para obtener la función validateEmail del archivo utils/helpers.ts
```

```
Usa trace_impact para analizar el impacto de modificar auth/login.ts:login
```
