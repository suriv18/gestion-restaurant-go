@echo off
cd /d "%~dp0"
where node >nul 2>&1
if errorlevel 1 (
  echo Instala Node.js LTS y vuelve a ejecutar este archivo.
  pause
  exit /b 1
)
echo Abre http://127.0.0.1:5173 en tu navegador.
echo Mantener esta ventana abierta durante el uso.
node servidor.mjs
pause
