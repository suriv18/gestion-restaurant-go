// Base temporal en memoria. No conecta ni altera un PostgreSQL externo.
import {createRequire} from 'node:module';
import {readFile,writeFile} from 'node:fs/promises';
const require=createRequire(process.env.RESTAURANTE_TEST_PACKAGE||import.meta.url);
const {PGlite}=require('@electric-sql/pglite');
const db=new PGlite();
try{
 const version=(await db.query('select version()')).rows[0].version;
 await db.exec(await readFile(new URL('../../02_base_datos/modelo_de_base_de_datos_para_restaurante.sql',import.meta.url),'utf8'));
 await db.exec(await readFile(new URL('./scenarios.sql',import.meta.url),'utf8'));
 await db.exec(await readFile(new URL('./audit_scenarios.sql',import.meta.url),'utf8'));
 const counts=(await db.query("select count(*)::int as tablas from information_schema.tables where table_schema='sch_restaurante' and table_type='BASE TABLE'")).rows[0];
 const result={estado:'CORRECTO',ejecutado_en:new Date().toISOString(),entorno:version,...counts,alcance:'DDL, escenarios funcionales, auditoría y baja lógica; base temporal WASM. No valida concurrencia multiproceso ni rendimiento de producción.'};
 await writeFile(new URL('./resultados_sql.json',import.meta.url),JSON.stringify(result,null,2));console.log(JSON.stringify(result,null,2));
}catch(e){console.error(e.message);process.exitCode=1;}finally{await db.close();}
