# ADR-005 — Baja de mesa con motivo

Estado: aceptado para este repositorio (2026-10-21). Aporta al ADR-03 del expediente (auditoría contextual y baja lógica selectiva). No debe confundirse con el ADR-05 del expediente («Integraciones externas recuperables»): la numeración del repositorio es independiente.

## Contexto

El expediente define `mesas` como catálogo de desactivación simple (`ACTIVIDAD_SIMPLE`): la ruta `POST /mesas/{id}/baja` «no recibe motivo» y RF-030 distingue 9 maestros con baja y motivo de 17 catálogos de desactivación simple. En la rebanada 1 de I2 (configuración y salón) el equipo decidió que dar de baja una mesa debe dejar constancia de quién, cuándo y por qué, igual que una sucursal. Una mesa retirada del salón es una decisión operativa que conviene explicar, y la restauración debe conservar el mismo ID.

## Decisión

- `mesas` pasa a ser el décimo maestro con baja y motivo. La migración `00008_configuracion_salon` agrega `eliminado_en`, `eliminado_por` y `motivo_eliminacion` con el mismo `CHECK` que `sucursales` (activa: los tres nulos; inactiva: los tres presentes y el motivo no vacío) y recrea el trigger `a0_metadatos` con `fn_metadatos_fila('true','true')`, que ya exige el motivo, fija actor y fecha de baja y los limpia al restaurar. `zz_auditoria` registra el evento completo.
- `POST /api/v1/mesas/{id}/baja` exige `{"motivo": "<1–2000 caracteres>"}`; sin motivo responde 422 `motivo_requerido`. La restauración conserva el ID y limpia los marcadores.
- La baja de zona sigue siendo simple y sin motivo; se rechaza con 409 `zona_con_mesas` si la zona tiene mesas activas.
- Una mesa dada de baja no se edita ni se habilita hasta restaurarla (409 `mesa_inactiva`). El trigger `a2_baja_operativa` (V4) impide dar de baja una mesa con pedido abierto.
- Las mesas existentes que estuvieran inactivas al migrar reciben el motivo `Baja anterior a la migración 00008`.

## Consecuencias

- Se corrigen los documentos del expediente que decían «No recibe motivo» o «9 maestros / 17 catálogos»: contrato (`openapi.yaml`), RF-030, ADR-03, `politica_campos_por_tabla.json`, `matriz_auditoria.html`, `diccionario_datos.html` y RES-CP-030. El SQL V4 y sus totales (49 tablas, 664 columnas) no se modifican: describen la base inicial y la diferencia vive en la migración 00008.
- Si el equipo revierte la decisión, basta omitir las columnas y el trigger de la migración y devolver la baja de mesa al contrato sin cuerpo.
- La interfaz (paso 4) pide el motivo al dar de baja una mesa, con el mismo diálogo que sucursales.

## Verificación

- Pruebas de integración de `tests/integration/atencion_mesas_test.go`: baja sin motivo → 422, motivo y actor en la fila y en `auditoria_eventos`, restauración con el mismo ID y reintento idempotente sin nuevo evento.
- `sha256sum -c SHA256SUMS.txt` en `docs/planificacion` tras actualizar el expediente.
