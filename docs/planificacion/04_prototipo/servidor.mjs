import {createServer} from 'node:http';
import {readFile,stat} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import {resolve,sep,extname} from 'node:path';
const root=fileURLToPath(new URL('./dist/',import.meta.url));
const types={'.html':'text/html; charset=utf-8','.js':'text/javascript; charset=utf-8','.css':'text/css; charset=utf-8','.svg':'image/svg+xml','.json':'application/json; charset=utf-8'};
const server=createServer(async(req,res)=>{try{
 if(!['GET','HEAD'].includes(req.method)){res.writeHead(405,{Allow:'GET, HEAD'}).end();return;}
 const pathname=decodeURIComponent(new URL(req.url,'http://localhost').pathname);
 const target=resolve(root,'.'+(pathname==='/'?'/index.html':pathname));
 if(!target.startsWith(root.endsWith(sep)?root:root+sep)){res.writeHead(403).end('No permitido');return;}
 if(!(await stat(target)).isFile()){res.writeHead(404).end('No encontrado');return;}
 const content=await readFile(target);res.writeHead(200,{'Content-Type':types[extname(target)]||'application/octet-stream','Cache-Control':'no-store','X-Content-Type-Options':'nosniff'});res.end(req.method==='HEAD'?undefined:content);
 }catch{res.writeHead(404).end('No encontrado');}});
server.on('error',error=>{console.error('No se pudo iniciar el prototipo:',error.message);process.exitCode=1;});
server.listen(5173,'127.0.0.1',()=>console.log('Prototipo: http://127.0.0.1:5173\nSolo datos de ejemplo. Ctrl+C para terminar.'));
