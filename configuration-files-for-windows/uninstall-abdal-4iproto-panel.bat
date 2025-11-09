@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM **********************************************************************
REM -------------------------------------------------------------------
REM Project Name : Abdal 4iProto Panel
REM File Name    : abdal-service-uninstall.bat
REM Author       : Ebrahim Shafiei (EbraSha)
REM Email        : Prof.Shafiei@Gmail.com
REM Created On   : 2025-11-09 19:37:56
REM Description  : Windows Service Uninstaller for Abdal 4iProto Panel and Server - Remove services and files with admin privileges
REM -------------------------------------------------------------------
REM
REM "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
REM – Ebrahim Shafiei
REM
REM **********************************************************************

:: Set console color scheme
color 0F

:: Service configuration
set "PANEL_SERVICE_NAME=Abdal4iProtoPanel"
set "SERVER_SERVICE_NAME=Abdal4iProtoServer"
set "PANEL_SERVICE_DISPLAY_NAME=Abdal 4iProto Panel"
set "SERVER_SERVICE_DISPLAY_NAME=Abdal 4iProto Server"
set "CURRENT_DIR=%CD%"
cd /d "%CURRENT_DIR%"
set "INSTALL_DIR=%CURRENT_DIR%"

:: Required files that may exist in current directory
set "REQUIRED_FILES=abdal-4iproto-panel.json abdal-4iproto-panel-windows.exe abdal-4iproto-server-windows.exe blocked_ips.json id_ed25519 id_ed25519.pub server_config.json users.json"

:: Check for admin privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    color 0C
    echo.
    echo ═══════════════════════════════════════════════════════════
    echo [ERROR] This script requires administrator privileges!
    echo.
    color 0E
    echo [INFO] Please run this script as Administrator.
    echo.
    color 0B
    echo [SOLUTION] Right-click on this file and select "Run as administrator"
    echo.
    echo ═══════════════════════════════════════════════════════════
    echo.
    color 0F
    pause
    exit /b 1
)

:: Check if services are installed
echo.
color 0B
echo [INFO] Checking if services are installed...
echo.

set "SERVICES_INSTALLED=0"
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    set "SERVICES_INSTALLED=1"
)

sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    set "SERVICES_INSTALLED=1"
)

if !SERVICES_INSTALLED! equ 0 (
    color 0E
    echo.
    echo ═══════════════════════════════════════════════════════════
    echo [WARNING] Services are not installed!
    echo.
    color 0B
    echo [INFO] Nothing to uninstall.
    echo.
    echo ═══════════════════════════════════════════════════════════
    echo.
    color 0F
    pause
    exit /b 0
)

color 0A
echo [SUCCESS] Services found!
echo.

:: Main menu
:MENU
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║          Abdal 4iProto Panel ^& Server UNINSTALLER           ║
echo ║                                                              ║
color 0E
echo ║  1. Remove Services Only                                     ║
color 0C
echo ║  2. Complete Removal (Services + Files)                     ║
color 0D
echo ║  3. Exit                                                      ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.
color 0F
set /p choice=Please select an option (1-3): 

if "%choice%"=="1" goto REMOVE_SERVICES_ONLY
if "%choice%"=="2" goto REMOVE_COMPLETE
if "%choice%"=="3" goto EXIT
color 0C
echo [ERROR] Invalid choice! Please select 1, 2, or 3.
color 0F
timeout /t 2 >nul
goto MENU

:REMOVE_SERVICES_ONLY
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0E
echo ║              REMOVE SERVICES ONLY                            ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.
color 0E
echo [WARNING] This will remove the following services:
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    echo   • %PANEL_SERVICE_NAME%
)
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    echo   • %SERVER_SERVICE_NAME%
)
echo.
color 0E
echo [WARNING] Files will remain in the installation directory.
echo.
color 0F
set /p confirm=Are you sure you want to continue? (yes/no): 

if /i not "!confirm!"=="yes" (
    color 0B
    echo.
    echo [INFO] Operation cancelled by user.
    echo.
    color 0F
    pause
    goto MENU
)

echo.
color 0B
echo [INFO] Starting service removal...
echo.

:: Stop and remove server service
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    color 0B
    echo [INFO] Stopping server service "%SERVER_SERVICE_NAME%"...
    sc stop "%SERVER_SERVICE_NAME%" >nul 2>&1
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Server service stopped successfully.
    ) else (
        color 0E
        echo [WARNING] Server service may already be stopped or failed to stop.
    )
    
    timeout /t 3 >nul
    
    color 0B
    echo [INFO] Removing server service "%SERVER_SERVICE_NAME%"...
    sc delete "%SERVER_SERVICE_NAME%" >nul 2>&1
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Server service removed successfully!
    ) else (
        color 0C
        echo [ERROR] Failed to remove server service!
    )
    echo.
)

:: Stop and remove panel service
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    color 0B
    echo [INFO] Stopping panel service "%PANEL_SERVICE_NAME%"...
    sc stop "%PANEL_SERVICE_NAME%" >nul 2>&1
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Panel service stopped successfully.
    ) else (
        color 0E
        echo [WARNING] Panel service may already be stopped or failed to stop.
    )
    
    timeout /t 3 >nul
    
    color 0B
    echo [INFO] Removing panel service "%PANEL_SERVICE_NAME%"...
    sc delete "%PANEL_SERVICE_NAME%" >nul 2>&1
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Panel service removed successfully!
    ) else (
        color 0C
        echo [ERROR] Failed to remove panel service!
    )
    echo.
)

color 0A
echo ═══════════════════════════════════════════════════════════
echo [SUCCESS] Services removed successfully!
echo ═══════════════════════════════════════════════════════════
echo.
color 0B
echo [INFO] Files are still in the installation directory: %INSTALL_DIR%
echo [INFO] You can manually delete this directory if needed.
echo.
color 0B
echo Programmer: Ebrahim Shafiei (EbraSha)
echo Email: Prof.Shafiei@Gmail.com
echo.
color 0F
pause
goto MENU

:REMOVE_COMPLETE
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0C
echo ║              COMPLETE REMOVAL                                ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.
color 0C
echo [WARNING] This will remove the following services:
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    echo   • %PANEL_SERVICE_NAME%
)
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    echo   • %SERVER_SERVICE_NAME%
)
echo.
color 0C
echo [WARNING] This will also DELETE ALL FILES in the installation directory:
echo   %INSTALL_DIR%
echo.
color 0C
echo [WARNING] The following files will be deleted:
for %%F in (%REQUIRED_FILES%) do (
    if exist "!INSTALL_DIR!\%%F" (
        echo   - %%F
    )
)
echo.
color 0C
echo [WARNING] This action cannot be undone!
echo.
color 0F
set /p confirm=Are you absolutely sure you want to continue? (yes/no): 

if /i not "!confirm!"=="yes" (
    color 0B
    echo.
    echo [INFO] Operation cancelled by user.
    echo.
    color 0F
    pause
    goto MENU
)

echo.
color 0B
echo [INFO] Starting complete removal...
echo.

:: Stop and remove server service
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    color 0B
    echo [INFO] Stopping server service "%SERVER_SERVICE_NAME%"...
    sc stop "%SERVER_SERVICE_NAME%" >nul 2>&1
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Server service stopped successfully.
    ) else (
        color 0E
        echo [WARNING] Server service may already be stopped or failed to stop.
    )
    
    timeout /t 3 >nul
    
    color 0B
    echo [INFO] Removing server service "%SERVER_SERVICE_NAME%"...
    sc delete "%SERVER_SERVICE_NAME%" >nul 2>&1
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Server service removed successfully!
    ) else (
        color 0C
        echo [ERROR] Failed to remove server service!
    )
    echo.
)

:: Stop and remove panel service
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    color 0B
    echo [INFO] Stopping panel service "%PANEL_SERVICE_NAME%"...
    sc stop "%PANEL_SERVICE_NAME%" >nul 2>&1
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Panel service stopped successfully.
    ) else (
        color 0E
        echo [WARNING] Panel service may already be stopped or failed to stop.
    )
    
    timeout /t 3 >nul
    
    color 0B
    echo [INFO] Removing panel service "%PANEL_SERVICE_NAME%"...
    sc delete "%PANEL_SERVICE_NAME%" >nul 2>&1
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Panel service removed successfully!
    ) else (
        color 0C
        echo [ERROR] Failed to remove panel service!
    )
    echo.
)

:: Delete files
color 0B
echo [INFO] Deleting files from installation directory...
echo.

set "FILES_DELETED=0"
for %%F in (%REQUIRED_FILES%) do (
    if exist "!INSTALL_DIR!\%%F" (
        del /F /Q "!INSTALL_DIR!\%%F" >nul 2>&1
        if %errorLevel% equ 0 (
            color 0A
            echo [SUCCESS] Deleted: %%F
            set "FILES_DELETED=1"
        ) else (
            color 0C
            echo [ERROR] Failed to delete: %%F
        )
    )
)

:: Try to remove directory if empty
if exist "%INSTALL_DIR%" (
    rmdir "%INSTALL_DIR%" >nul 2>&1
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Removed installation directory: %INSTALL_DIR%
    ) else (
        color 0E
        echo [WARNING] Installation directory could not be removed (may contain other files).
        echo [INFO] Directory: %INSTALL_DIR%
    )
)

echo.
color 0A
echo ═══════════════════════════════════════════════════════════
echo [SUCCESS] Complete removal finished!
echo ═══════════════════════════════════════════════════════════
echo.
color 0B
echo Programmer: Ebrahim Shafiei (EbraSha)
echo Email: Prof.Shafiei@Gmail.com
echo.
color 0F
pause
goto MENU

:EXIT
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0A
echo ║                      THANK YOU!                             ║
color 0B
echo ║                                                              ║
color 0F
echo ║  Abdal 4iProto Panel ^& Server Uninstaller                   ║
echo ║  Developed by: Ebrahim Shafiei (EbraSha)                    ║
echo ║  Email: Prof.Shafiei@Gmail.com                               ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.
color 0A
echo [INFO] Goodbye! Have a great day!
echo.
color 0F
pause
exit /b 0

