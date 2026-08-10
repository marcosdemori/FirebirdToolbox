@echo off
setlocal EnableExtensions
chcp 65001 >nul
cd /d "%~dp0"

echo ============================================================
echo Firebird Toolbox - Publicar projeto no GitHub
echo ============================================================
echo.

where git >nul 2>nul
if errorlevel 1 (
    echo ERRO: Git nao foi encontrado no PATH.
    echo Instale o Git for Windows e execute novamente.
    pause
    exit /b 1
)

if not exist ".git" (
    echo Inicializando repositorio Git...
    git init || goto :erro
)

git branch -M main || goto :erro

echo.
echo Adicionando arquivos permitidos pelo .gitignore...
git add . || goto :erro

echo.
echo ============================================================
echo CONFIRA ABAIXO O QUE SERA VERSIONADO
echo ============================================================
git status --short

echo.
echo IMPORTANTE: nao continue se aparecer banco, backup, RECOVERY,
echo senha privada, certificado ou qualquer dado de cliente.
echo.
set /p CONFIRMA=Continuar com o commit? [S/N]:
if /I not "%CONFIRMA%"=="S" (
    echo Operacao cancelada.
    pause
    exit /b 0
)

git diff --cached --quiet
if not errorlevel 1 (
    echo Nao ha alteracoes novas para commit.
) else (
    git commit -m "fix(release): corrigir build Windows no GitHub Actions v1.2.11" || goto :erro
)

echo.
set /p REPOURL=Cole a URL HTTPS do repositorio GitHub:
if "%REPOURL%"=="" (
    echo URL nao informada. O commit local foi mantido.
    pause
    exit /b 0
)

git remote get-url origin >nul 2>nul
if errorlevel 1 (
    git remote add origin "%REPOURL%" || goto :erro
) else (
    git remote set-url origin "%REPOURL%" || goto :erro
)

echo.
echo Enviando branch main...
git push -u origin main || goto :erro

echo.
set /p TAG=Criar e enviar a tag v1.2.11 para gerar a Release automatica? [S/N]:
if /I "%TAG%"=="S" (
    git rev-parse v1.2.11 >nul 2>nul
    if errorlevel 1 (
        git tag -a v1.2.11 -m "Firebird Toolbox v1.2.11" || goto :erro
    )
    git push origin v1.2.11 || goto :erro
    echo.
    echo Tag enviada. O GitHub Actions deve iniciar o workflow Release.
)

echo.
echo ============================================================
echo Projeto publicado com sucesso.
echo ============================================================
pause
exit /b 0

:erro
echo.
echo ERRO: o Git retornou uma falha. Revise a mensagem acima.
pause
exit /b 1
