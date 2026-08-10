@echo off
SETLOCAL ENABLEEXTENSIONS

:: ==========================================
:: Script de Manutenção Avançada Firebird 5.0
:: ==========================================

:: Configurações de ambiente
SET ISC_USER=SYSDBA
SET ISC_PASSWORD=masterkey

:: Caminhos do banco
SET DB_PATH=C:\Users\Usuario\Desktop\Uteis\Bats\Banco de dados
SET DB_FILE=DBCOM.FDB
SET DB_FULL_PATH=%DB_PATH%\%DB_FILE%
SET RESTORE_PATH=%DB_PATH%\DBCOM50.RED

:: Caminho do Firebird 5.0
SET FIREBIRD_PATH=C:\Program Files\Firebird\Firebird_5_0
SET GFIX="%FIREBIRD_PATH%\gfix.exe"
SET GBAK="%FIREBIRD_PATH%\gbak.exe"

:: Serviço Firebird 5.0
SET FIREBIRD_SERVICE=Firebird5.0

:: Nome do backup com timestamp
FOR /F %%I IN ('powershell -command "Get-Date -Format yyyyMMdd_HHmmss"') DO SET TS=%%I
SET BACKUP_FILE="%DB_PATH%\DBCOM_%TS%.BAK"

:: Caminho TCP/IP para o banco
SET DB_TCP=127.0.0.1/3055:"%DB_FULL_PATH%"

:: Verificações
IF NOT EXIST %GFIX% (
    echo [ERRO] GFIX nao encontrado.
    pause
    exit /b
)

IF NOT EXIST %GBAK% (
    echo [ERRO] GBAK nao encontrado.
    pause
    exit /b
)

FOR %%F IN ("%DB_FILE%") DO SET SIZE=%%~zF
SET /A MB_SIZE=%SIZE% / 1048576
SET /A REMAINDER=(%SIZE% %% 1048576) * 100 / 1048576

echo =====================================================
echo Iniciando manutencao do banco de dados
echo Tamanho: %MB_SIZE%.%REMAINDER% MB
echo =====================================================
pause

:: Validação e Sweep
echo =====================================================
echo Validando banco...
echo =====================================================
%GFIX% -user %ISC_USER% -pass %ISC_PASSWORD% -v -full %DB_TCP% || GOTO :erro_validar

echo =====================================================
echo Executando Sweep...
echo =====================================================
%GFIX% -user %ISC_USER% -pass %ISC_PASSWORD% -sweep %DB_TCP% || GOTO :erro_sweep

:: Backup
echo =====================================================
echo Realizando backup...
echo Arquivo: %BACKUP_FILE%
echo =====================================================
%GBAK% -b -v -user %ISC_USER% -pass %ISC_PASSWORD% %DB_TCP% %BACKUP_FILE% || GOTO :erro_backup

IF NOT EXIST %BACKUP_FILE% (
    echo [ERRO] Backup não gerado: %BACKUP_FILE%
    pause
    exit /b
)

:: Parando Firebird
echo =====================================================
echo Parando serviço do Firebird...
echo =====================================================
net stop %FIREBIRD_SERVICE%
timeout /t 2 >nul

echo Finalizando processos Firebird...
taskkill /F /IM fbserver.exe >nul 2>&1
taskkill /F /IM fb_inet_server.exe >nul 2>&1
taskkill /F /IM fbguard.exe >nul 2>&1

:: Aguarda até que Firebird esteja realmente parado
:aguarda_fim
tasklist | findstr /i "fbserver.exe fb_inet_server.exe fbguard.exe" >nul
IF NOT ERRORLEVEL 1 (
    echo Aguardando fechamento completo do Firebird...
    timeout /t 2 >nul
    goto aguarda_fim
)

:: Restauração usando Service Manager para evitar erro de memória compartilhada
echo =====================================================
echo Restaurando banco de dados com Service Manager...
echo =====================================================
%GBAK% -c -v -se localhost:service_mgr -user %ISC_USER% -pass %ISC_PASSWORD% -REP %BACKUP_FILE% "%RESTORE_PATH%" || GOTO :erro_restore

IF EXIST "%RESTORE_PATH%" (
    echo Banco restaurado em: %RESTORE_PATH%
) ELSE (
    echo [ATENÇÃO] Banco restaurado pode ter sido salvo em outro diretório!
)

:: Inicia Firebird
echo =====================================================
echo Iniciando o serviço do Firebird...
echo =====================================================
net start %FIREBIRD_SERVICE%
timeout /t 2 >nul

echo =====================================================
echo Processo finalizado com sucesso!
echo =====================================================
pause
exit /b

:: ERROS
:erro_validar
echo [ERRO] na validacao.
pause
exit /b

:erro_sweep
echo [ERRO] ao executar sweep.
pause
exit /b

:erro_backup
echo [ERRO] ao gerar backup.
pause
exit /b

:erro_restore
echo [ERRO] ao restaurar banco.
pause
exit /b
