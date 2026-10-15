@echo off
setlocal EnableExtensions
chcp 65001 >nul
cd /d "%~dp0"

:menu
cls
echo ============================================================
echo Firebird Toolbox - Painel do projeto
echo ============================================================
echo.
echo   1 - Gerar o executavel em dist\
echo   2 - Publicar o projeto no GitHub
echo   3 - Gerar o executavel e depois publicar
echo   0 - Sair
echo.
set "OPCAO="
set /p OPCAO=Escolha uma opcao [1/2/3/0]:

if "%OPCAO%"=="1" goto :somente_build
if "%OPCAO%"=="2" goto :somente_publicar
if "%OPCAO%"=="3" goto :build_e_publicar
if "%OPCAO%"=="0" exit /b 0
rem Sem opcao (Enter vazio ou entrada redirecionada) o painel encerra em vez
rem de repetir o menu para sempre.
if not defined OPCAO exit /b 0
echo.
echo Opcao invalida.
pause
goto :menu

:somente_build
call :build || goto :erro_final
echo.
pause
exit /b 0

:build_e_publicar
call :build || goto :erro_final
echo.
pause
goto :publicar

:somente_publicar
goto :publicar

rem ------------------------------------------------------------
rem Build: tools\build.ps1 e a unica forma suportada de gerar o
rem executavel. Ele recria dist\ do zero e grava la o aplicativo,
rem os binarios de teste, o SHA256.txt e o SIGNATURE_STATUS.txt.
rem A raiz do repositorio nunca recebe executavel.
rem ------------------------------------------------------------
:build
echo.
echo ============================================================
echo Gerando o executavel em dist\
echo ============================================================
echo.

call :garantir_go
if errorlevel 1 exit /b 1

where powershell >nul 2>nul
if errorlevel 1 (
    echo ERRO: Windows PowerShell nao foi encontrado no PATH.
    exit /b 1
)

echo Este build local sai SEM assinatura Authenticode.
echo Ele serve apenas para desenvolvimento e teste: NAO DISTRIBUIR.
echo A Release oficial e assinada pelo Microsoft Artifact Signing no GitHub Actions.
echo.
set "CONFIRMABUILD="
set /p CONFIRMABUILD=Gerar o build local sem assinatura? [S/N]:
if /I not "%CONFIRMABUILD%"=="S" (
    echo Build cancelado.
    exit /b 1
)

echo.
if not exist "%~dp0tools\build.ps1" (
    echo ERRO: "%~dp0tools\build.ps1" nao foi encontrado.
    exit /b 1
)

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0tools\build.ps1" -Version "1.2.32" -AllowUnsignedDev
if errorlevel 1 (
    echo.
    echo ERRO: o build falhou. Revise a mensagem acima.
    exit /b 1
)

rem Confere o artefato em vez de confiar apenas no codigo de saida: um build
rem que termina em zero sem gravar o executavel nao pode passar por concluido.
if not exist "%~dp0dist\FirebirdToolbox_v1.2.32_x86_x64*.exe" (
    echo.
    echo ERRO: o build terminou sem gravar o executavel em dist\.
    exit /b 1
)

echo.
echo ============================================================
echo Conteudo de dist\
echo ============================================================
dir /b "%~dp0dist"
exit /b 0

rem ------------------------------------------------------------
rem O Go costuma ser instalado fora do PATH. Acrescenta o caminho
rem padrao somente nesta sessao do .bat, sem alterar o ambiente.
rem ------------------------------------------------------------
:garantir_go
where go >nul 2>nul
if not errorlevel 1 exit /b 0
if exist "%ProgramFiles%\Go\bin\go.exe" (
    set "PATH=%ProgramFiles%\Go\bin;%PATH%"
    exit /b 0
)
if exist "%LOCALAPPDATA%\Programs\Go\bin\go.exe" (
    set "PATH=%LOCALAPPDATA%\Programs\Go\bin;%PATH%"
    exit /b 0
)
echo ERRO: Go nao foi encontrado no PATH nem em "%ProgramFiles%\Go\bin".
echo Instale o Go ou acrescente a pasta bin dele ao PATH.
exit /b 1

:publicar
echo.
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
    git commit -m "fix(recovery): restore com -n quando a FK nao pode ser recriada v1.2.32" || goto :erro
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
echo.
echo ATENCAO: a v1.2.32 NAO publica sem Microsoft Artifact Signing.
echo Configure os Secrets/Variables descritos em docs\DISTRIBUICAO_WINDOWS.md antes de criar a tag.
echo.
set /p TAG=Criar e enviar a tag v1.2.32 para gerar a Release automatica? [S/N]:
if /I "%TAG%"=="S" (
    git rev-parse v1.2.32 >nul 2>nul
    if errorlevel 1 (
        git tag -a v1.2.32 -m "Firebird Toolbox v1.2.32" || goto :erro
    )
    git push origin v1.2.32 || goto :erro
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

:erro_final
echo.
pause
exit /b 1
