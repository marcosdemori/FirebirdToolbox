@echo off
SETLOCAL ENABLEEXTENSIONS

:: ================================================
:: RESTAURAÇÃO DE BACKUP - FIREBIRD 3.0
:: ================================================

:: CONFIGURAÇÕES
SET FIREBIRD_PATH=C:\Program Files\Firebird\Firebird_3_0
SET GBAK="%FIREBIRD_PATH%\gbak.exe"

SET BACKUP_FILE=C:\Users\Usuario\Desktop\Uteis\Bats\Banco de dados\DBCOM_20260122_084943.BAK
SET DEST_DB=C:\Users\Usuario\Desktop\Uteis\Bats\Banco de dados\DBCOM30.RED

SET ISC_USER=SYSDBA
SET ISC_PASSWORD=masterkey

:: MOSTRA INÍCIO
ECHO.
ECHO =====================================================
ECHO Restaurando backup do Firebird 3.0...
ECHO Backup  : %BACKUP_FILE%
ECHO Destino : %DEST_DB%
ECHO Início  : %TIME%
ECHO =====================================================
ECHO.

:: EXECUTA RESTAURAÇÃO USANDO SERVICE MANAGER
%GBAK% -c -se localhost:service_mgr -user %ISC_USER% -password %ISC_PASSWORD% ^
       "%BACKUP_FILE%" "%DEST_DB%" -v

:: VERIFICA ERRO DE RETORNO
IF %ERRORLEVEL% NEQ 0 (
    ECHO.
    ECHO [ERRO] durante a restauração. Código: %ERRORLEVEL%
    ECHO Fim: %TIME%
    ECHO =====================================================
    PAUSE
    EXIT /B %ERRORLEVEL%
)

:: FINALIZA COM SUCESSO
ECHO.
ECHO =====================================================
ECHO Restauração concluída com sucesso.
ECHO Fim: %TIME%
ECHO =====================================================
PAUSE
ENDLOCAL
EXIT /B 0
