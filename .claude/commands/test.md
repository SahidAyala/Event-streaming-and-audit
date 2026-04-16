---
description: Generate or run unit tests for modified modules
---

Genera o ejecuta unit tests para los cambios actuales del proyecto.

## Paso 1: Detectar archivos modificados

Ejecuta:


git status --short


Extrae los archivos modificados dentro de:


src/contexts/


Ignora:


*.spec.ts
*.e2e-spec.ts
*.dto.ts

---

## Paso 2: Verificar si existen tests

Para cada archivo modificado:

1. Detecta su test correspondiente:

Ejemplo:


src/contexts/compliance/application/compliance.service.ts


Test esperado:


src/contexts/compliance/application/compliance.service.spec.ts


Si el test **no existe**, créalo.

---

## Paso 3: Generar tests

Genera tests siguiendo las convenciones del proyecto.

### Reglas

Usar:


Vitest


Estructura típica:


describe('ComplianceService', () => {
describe('createComplianceRequest', () => {
it('should create compliance request', () => {})
})
})


### Buenas prácticas

Los tests deben:

- mockear repositorios
- probar lógica de negocio
- cubrir casos:


happy path
validation errors
edge cases


---

## Paso 4: Ejecutar tests

Ejecuta:


pnpm test


o si el proyecto usa npm:


npm run test


Si hay errores:

- mostrar errores
- detener proceso

---

## Paso 5: Mostrar cobertura

Si existe coverage:


pnpm test:cov


Mostrar resumen.

---

## Paso 6: Confirmar

Mostrar resultado final:


Tests passed
Tests generated
Coverage summary


---

## Reglas importantes

- No modificar código de producción
- Solo generar o actualizar tests
- Mantener mocks consistentes con arquitectura DDD
- No agregar Co-Authored-By