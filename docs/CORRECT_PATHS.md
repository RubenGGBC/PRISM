# 🎯 Rutas Correctas para Usar con Claude

## ✅ Archivos Indexados

La base de datos tiene estos archivos del test/sample-repo:

- `auth\login.ts`
- `db\user.ts`
- `utils\helpers.ts`

## 📋 Comandos que FUNCIONARÁN

### 1. Listar funciones en auth/login.ts:
```
Usa list_functions con file="auth\login.ts"
```

### 2. Obtener función específica:
```
Usa get_file_smart con file="auth\login.ts" y symbol="login"
```

### 3. Buscar funciones:
```
Usa search_context para buscar "password"
```

### 4. Ver funciones en helpers:
```
Usa list_functions con file="utils\helpers.ts"
```

### 5. Análisis de impacto:
```
Usa trace_impact con function_id="auth\login.ts:login"
```

## 🧪 Prueba AHORA

En la Terminal 2 con Claude, escribe:

```
Usa get_file_smart con file="auth\login.ts" y symbol="login"
```

O prueba:

```
Usa list_functions con file="auth\login.ts"
```

## ⚠️ Nota sobre las rutas

- Usa barras invertidas: `auth\login.ts`
- NO uses: `test/sample-repo/auth/login.ts`
- Las rutas son relativas al directorio indexado

## 📊 Funciones Disponibles

### auth\login.ts:
- login
- logout  
- createSession
- deleteSession

### db\user.ts:
- getUser
- updateUser
- deleteUser
- listUsers

### utils\helpers.ts:
- hashPassword
- validateEmail
- generateToken
- formatDate
