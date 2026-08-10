@echo off
SETLOCAL ENABLEEXTENSIONS

:: ================================================
:: RESTAURAÇÃO DE BACKUP - FIREBIRD 5.0
:: ================================================

:: CONFIGURAÇÕES
SET FIREBIRD_PATH=C:\Program Files\Firebird\Firebird_5_0
SET GBAK="%FIREBIRD_PATH%\gbak.exe"
SET GFIX="%FIREBIRD_PATH%\gfix.exe"
SET GSTAT="%FIREBIRD_PATH%\gstat.exe"

SET BACKUP_FILE=C:\Users\Usuario\Desktop\Uteis\Bats\Banco de dados\DBCOM_20260805_174405.BAK
SET DEST_DB=C:\Users\Usuario\Desktop\Uteis\Bats\Banco de dados\DBCOM50.RED

SET ISC_USER=SYSDBA
SET ISC_PASSWORD=masterkey

:: MOSTRA INÍCIO
ECHO.
ECHO =====================================================
ECHO Restaurando backup do Firebird 5.0...
ECHO Backup  : %BACKUP_FILE%
ECHO Destino : %DEST_DB%
ECHO Inicio  : %TIME%
ECHO =====================================================
ECHO.

:: 1) RESTAURAÇÃO (sem sql_dialect - o gbak não suporta)
%GBAK% -c -se localhost:service_mgr -user %ISC_USER% -password %ISC_PASSWORD% ^
       "%BACKUP_FILE%" "%DEST_DB%" -v
IF %ERRORLEVEL% NEQ 0 (
    ECHO.
    ECHO [ERRO] durante a restauracao. Codigo: %ERRORLEVEL%
    ECHO Fim: %TIME%
    ECHO =====================================================
    PAUSE
    EXIT /B %ERRORLEVEL%
)

:: 2) AJUSTA O DIALETO PARA 1 COM GFIX
ECHO.
ECHO Ajustando dialeto para 1 com gfix...
%GFIX% -user %ISC_USER% -password %ISC_PASSWORD% -sql_dialect 1 "%DEST_DB%"
IF %ERRORLEVEL% NEQ 0 (
    ECHO.
    ECHO [ERRO] ao ajustar dialeto com gfix. Codigo: %ERRORLEVEL%
    ECHO Fim: %TIME%
    ECHO =====================================================
    PAUSE
    EXIT /B %ERRORLEVEL%
)

:: (Opcional) 3) MOSTRA O HEADER PARA CONFERIR O DIALETO
ECHO.
ECHO Conferindo header (procure por "ODS" e "dialect")...
%GSTAT% -h "%DEST_DB%" | findstr /I /C:"dialect" /C:"ODS"

:: FINALIZA COM SUCESSO
ECHO.
ECHO =====================================================
ECHO Restauracao concluida e dialeto ajustado para 1.
ECHO Fim: %TIME%
ECHO =====================================================
PAUSE
ENDLOCAL
EXIT /B 0
