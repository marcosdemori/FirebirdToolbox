@echo off
SETLOCAL ENABLEEXTENSIONS

:: ==========================================
:: Script de Manutencao Avancada Firebird 3.0
:: ==========================================

:: Configuracoes de ambiente
SET ISC_USER=SYSDBA
SET ISC_PASSWORD=masterkey

:: Caminhos do banco
SET DB_PATH=C:\Users\Usuario\Desktop\Uteis\Bats\Banco de dados
SET DB_FILE=DBCOM.FDB
SET DB_FULL_PATH=%DB_PATH%\%DB_FILE%
SET RESTORE_PATH=%DB_PATH%\DBCOM30.RED

:: Caminho do Firebird 3.0
SET FIREBIRD_PATH=C:\Program Files\Firebird\Firebird_3_0
SET GFIX="%FIREBIRD_PATH%\gfix.exe"
SET GBAK="%FIREBIRD_PATH%\gbak.exe"

:: Servico Firebird 3.0
SET FIREBIRD_SERVICE=Firebird3.0

:: Nome do backup com timestamp
FOR /F %%I IN ('powershell -NoProfile -Command "Get-Date -Format yyyyMMdd_HHmmss"') DO SET TS=%%I
SET BACKUP_FILE="%DB_PATH%\DBCOM_%TS%.BAK"

:: Caminho TCP/IP para o banco
SET DB_TCP=127.0.0.1/3053:"%DB_FULL_PATH%"

:: Verificacoes
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

IF NOT EXIST "%DB_FULL_PATH%" (
    echo [ERRO] Banco nao encontrado: "%DB_FULL_PATH%"
    pause
    exit /b
)

FOR /F "usebackq delims=" %%S IN (`powershell -NoProfile -Command "$f='%DB_FULL_PATH%'; if (Test-Path -LiteralPath $f) { [math]::Round((Get-Item -LiteralPath $f).Length / 1MB, 2) } else { '0' }"`) DO SET DB_SIZE_MB=%%S

echo =====================================================
echo Iniciando manutencao do banco de dados
echo Tamanho: %DB_SIZE_MB% MB
echo =====================================================
pause

:: Validacao e Sweep
echo =====================================================
echo Validando banco...
echo =====================================================
%GFIX% -user %ISC_USER% -pass %ISC_PASSWORD% -v -full %DB_TCP% || GOTO :erro_validar

:continua_pos_validacao
echo =====================================================
echo Executando Sweep...
echo =====================================================
%GFIX% -user %ISC_USER% -pass %ISC_PASSWORD% -sweep %DB_TCP% || GOTO :erro_sweep

:: Backup
echo =====================================================
echo Realizando backup...
echo Arquivo: %BACKUP_FILE%
echo =====================================================
%GBAK% -b -g -v -user %ISC_USER% -pass %ISC_PASSWORD% %DB_TCP% %BACKUP_FILE% || GOTO :erro_backup

IF NOT EXIST %BACKUP_FILE% (
    echo [ERRO] Backup nao gerado: %BACKUP_FILE%
    pause
    exit /b
)

:parar_servico_para_restore

:: Parando Firebird
echo =====================================================
echo Parando servico do Firebird...
echo =====================================================
net stop %FIREBIRD_SERVICE%
timeout /t 2 >nul

echo Finalizando processos Firebird...
taskkill /F /IM fbserver.exe >nul 2>&1
taskkill /F /IM fb_inet_server.exe >nul 2>&1
taskkill /F /IM fbguard.exe >nul 2>&1

:: Aguarda ate que Firebird esteja realmente parado
:aguarda_fim
tasklist | findstr /i "fbserver.exe fb_inet_server.exe fbguard.exe" >nul
IF NOT ERRORLEVEL 1 (
    echo Aguardando fechamento completo do Firebird...
    timeout /t 2 >nul
    goto aguarda_fim
)

:: Restauracao usando Service Manager para evitar erro de memoria compartilhada
echo =====================================================
echo Restaurando banco de dados com Service Manager...
echo =====================================================
%GBAK% -c -v -se localhost:service_mgr -user %ISC_USER% -pass %ISC_PASSWORD% -REP %BACKUP_FILE% "%RESTORE_PATH%" || GOTO :erro_restore

IF EXIST "%RESTORE_PATH%" (
    echo Banco restaurado em: %RESTORE_PATH%
) ELSE (
    echo [ATENCAO] Banco restaurado pode ter sido salvo em outro diretorio!
)

:: Inicia Firebird
echo =====================================================
echo Iniciando o servico do Firebird...
echo =====================================================
net start %FIREBIRD_SERVICE%
timeout /t 2 >nul

echo =====================================================
echo Validando banco restaurado...
echo =====================================================
SET DB_RESTORE_TCP=127.0.0.1/3053:"%RESTORE_PATH%"
%GFIX% -user %ISC_USER% -pass %ISC_PASSWORD% -v -full %DB_RESTORE_TCP%

echo =====================================================
echo Processo finalizado!
echo Banco original: %DB_FULL_PATH%
echo Backup gerado: %BACKUP_FILE%
echo Banco restaurado: %RESTORE_PATH%
echo =====================================================
pause
exit /b

:: ERROS
:erro_validar
echo [ERRO] na validacao.
echo.
echo O erro pode ser corrupcao, como BLOB not found.
echo O reparo com gfix -mend pode permitir recuperar o banco,
echo mas pode descartar paginas/registros corrompidos.
echo.
echo RECOMENDADO: execute este BAT em uma copia do banco, nao no original.
echo.
choice /C SN /M "Deseja tentar reparo com gfix -mend agora"
IF ERRORLEVEL 2 GOTO :fim_com_erro_validacao
IF ERRORLEVEL 1 GOTO :executar_mend

:executar_mend
echo =====================================================
echo Tentando reparo com gfix -mend -full...
echo =====================================================
%GFIX% -user %ISC_USER% -pass %ISC_PASSWORD% -mend -full %DB_TCP% || GOTO :erro_mend

echo =====================================================
echo Validando novamente apos gfix -mend...
echo =====================================================
%GFIX% -user %ISC_USER% -pass %ISC_PASSWORD% -v -full %DB_TCP%

echo.
echo Mesmo que ainda aparecam avisos na validacao, o script tentara gerar backup e restaurar.
echo Se o backup falhar, sera necessario localizar manualmente o registro corrompido na tabela DOCIT2.
echo.
pause
GOTO :continua_pos_validacao

:erro_mend
echo [ERRO] ao executar gfix -mend.
pause
exit /b

:fim_com_erro_validacao
echo Reparo cancelado pelo usuario.
pause
exit /b

:erro_sweep
echo [ERRO] ao executar sweep.
echo.
echo Continuar sem sweep pode ser necessario em banco corrompido.
choice /C SN /M "Deseja tentar continuar para backup mesmo assim"
IF ERRORLEVEL 2 exit /b
IF ERRORLEVEL 1 GOTO :backup_sem_sweep

:backup_sem_sweep
echo =====================================================
echo Realizando backup sem sweep...
echo Arquivo: %BACKUP_FILE%
echo =====================================================
%GBAK% -b -g -v -user %ISC_USER% -pass %ISC_PASSWORD% %DB_TCP% %BACKUP_FILE% || GOTO :erro_backup
GOTO :apos_backup_sem_sweep

:apos_backup_sem_sweep
IF NOT EXIST %BACKUP_FILE% (
    echo [ERRO] Backup nao gerado: %BACKUP_FILE%
    pause
    exit /b
)
GOTO :parar_servico_para_restore

:erro_backup
echo [ERRO] ao gerar backup.
echo.
echo Se aparecer BLOB not found aqui, o reparo automatico nao foi suficiente.
echo O proximo passo e localizar o registro corrompido na tabela DOCIT2.
pause
exit /b

:erro_restore
echo [ERRO] ao restaurar banco.
pause
exit /b
