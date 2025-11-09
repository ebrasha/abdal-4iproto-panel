@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM **********************************************************************
REM -------------------------------------------------------------------
REM Project Name : Abdal 4iProto Panel
REM File Name    : abdal-service-manager.bat
REM Author       : Ebrahim Shafiei (EbraSha)
REM Email        : Prof.Shafiei@Gmail.com
REM Created On   : 2025-11-09 19:30:23
REM Description  : Windows Service Manager for Abdal 4iProto Panel and Server - Install/Uninstall services with admin privileges
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
set "PANEL_SERVICE_DESCRIPTION=Abdal 4iProto Panel - Advanced Network Protocol Management Panel"
set "SERVER_SERVICE_DESCRIPTION=Abdal 4iProto Server - Advanced Network Protocol Server"
set "PANEL_EXECUTABLE=abdal-4iproto-panel-windows.exe"
set "SERVER_EXECUTABLE=abdal-4iproto-server-windows.exe"
set "CURRENT_DIR=%CD%"
cd /d "%CURRENT_DIR%"
set "INSTALL_DIR=%CURRENT_DIR%"

:: Required files that must exist in current directory
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

:: Check if services are already installed
echo.
color 0B
echo [INFO] Checking if services are already installed...
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

if !SERVICES_INSTALLED! equ 1 (
    color 0E
    echo.
    echo ═══════════════════════════════════════════════════════════
    echo [WARNING] Services are already installed!
    echo.
    color 0E
    echo [INFO] The following services are already installed:
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
    echo [INFO] If you want to reinstall, please uninstall first using option 2.
    echo.
    echo ═══════════════════════════════════════════════════════════
    echo.
    color 0F
    pause
    exit /b 0
)

goto MAIN_MENU

:MAIN_MENU
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║          Abdal 4iProto Panel ^& Server MANAGER              ║
echo ║                                                              ║
color 0A
echo ║  1. Install Services                                        ║
color 0C
echo ║  2. Stop and Remove Services                                ║
color 0E
echo ║  3. Check Service Status                                     ║
color 0D
echo ║  4. Exit                                                     ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.
color 0F
set /p choice=Please select an option (1-4): 

if "%choice%"=="1" goto INSTALL_SERVICE
if "%choice%"=="2" goto REMOVE_SERVICE
if "%choice%"=="3" goto CHECK_STATUS
if "%choice%"=="4" goto EXIT
color 0C
echo [ERROR] Invalid choice! Please select 1, 2, 3, or 4.
color 0F
timeout /t 2 >nul
goto MAIN_MENU

:INSTALL_SERVICE
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0A
echo ║                  INSTALLING SERVICES                         ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.

:: Check if all required files exist
color 0B
echo [INFO] Checking required files...
echo.

set "MISSING_FILES=0"
for %%F in (%REQUIRED_FILES%) do (
    if not exist "!CURRENT_DIR!\%%F" (
        color 0C
        echo [ERROR] Missing file: %%F
        set "MISSING_FILES=1"
    ) else (
        color 0A
        echo [SUCCESS] Found: %%F
    )
)

if !MISSING_FILES! equ 1 (
    echo.
    color 0C
    echo [ERROR] Some required files are missing!
    echo.
    color 0E
    echo [INFO] Please make sure all required files are in the current directory:
    echo   !CURRENT_DIR!
    echo.
    echo [INFO] Required files:
    for %%F in (%REQUIRED_FILES%) do (
        echo   - %%F
    )
    echo.
    color 0F
    pause
    goto MAIN_MENU
)

echo.
color 0A
echo [SUCCESS] All required files found!
echo.

:: Check if services already exist
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    color 0E
    echo [WARNING] Service "%SERVER_SERVICE_NAME%" already exists!
    color 0B
    echo [INFO] Stopping and removing existing service...
    
    sc stop "%SERVER_SERVICE_NAME%" >nul 2>&1
    timeout /t 3 >nul
    
    sc delete "%SERVER_SERVICE_NAME%" >nul 2>&1
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Existing server service removed successfully.
    ) else (
        color 0C
        echo [ERROR] Failed to remove existing server service.
        echo.
        color 0F
        pause
        goto MAIN_MENU
    )
    echo.
)

sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    color 0E
    echo [WARNING] Service "%PANEL_SERVICE_NAME%" already exists!
    color 0B
    echo [INFO] Stopping and removing existing service...
    
    sc stop "%PANEL_SERVICE_NAME%" >nul 2>&1
    timeout /t 3 >nul
    
    sc delete "%PANEL_SERVICE_NAME%" >nul 2>&1
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Existing panel service removed successfully.
    ) else (
        color 0C
        echo [ERROR] Failed to remove existing panel service.
        echo.
        color 0F
        pause
        goto MAIN_MENU
    )
    echo.
)

:: Installation directory is the current directory
color 0B
echo [INFO] Installation directory: %INSTALL_DIR%
color 0A
echo [SUCCESS] Files are already in the installation directory
echo.

:: Create server service
color 0B
echo [INFO] Installing server service "%SERVER_SERVICE_NAME%"...
sc create "%SERVER_SERVICE_NAME%" binPath= "\"%INSTALL_DIR%\%SERVER_EXECUTABLE%\"" DisplayName= "%SERVER_SERVICE_DISPLAY_NAME%" start= auto

if %errorLevel% equ 0 (
    color 0A
    echo [SUCCESS] Server service created successfully!
    sc description "%SERVER_SERVICE_NAME%" "%SERVER_SERVICE_DESCRIPTION%"
    
    color 0B
    echo [INFO] Starting server service...
    sc start "%SERVER_SERVICE_NAME%"
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Server service started successfully!
    ) else (
        color 0E
        echo [WARNING] Server service created but failed to start.
        color 0B
        echo [INFO] You can start it manually from Services.msc
    )
) else (
    color 0C
    echo [ERROR] Failed to create server service!
    color 0E
    echo [INFO] Please check the executable path and permissions.
)
echo.

:: Create panel service
color 0B
echo [INFO] Installing panel service "%PANEL_SERVICE_NAME%"...
sc create "%PANEL_SERVICE_NAME%" binPath= "\"%INSTALL_DIR%\%PANEL_EXECUTABLE%\"" DisplayName= "%PANEL_SERVICE_DISPLAY_NAME%" start= auto

if %errorLevel% equ 0 (
    color 0A
    echo [SUCCESS] Panel service created successfully!
    sc description "%PANEL_SERVICE_NAME%" "%PANEL_SERVICE_DESCRIPTION%"
    
    color 0B
    echo [INFO] Starting panel service...
    sc start "%PANEL_SERVICE_NAME%"
    
    if %errorLevel% equ 0 (
        color 0A
        echo [SUCCESS] Panel service started successfully!
    ) else (
        color 0E
        echo [WARNING] Panel service created but failed to start.
        color 0B
        echo [INFO] You can start it manually from Services.msc
    )
) else (
    color 0C
    echo [ERROR] Failed to create panel service!
    color 0E
    echo [INFO] Please check the executable path and permissions.
)
echo.

color 0A
echo [COMPLETE] Abdal 4iProto Panel and Server are now running as Windows services.
color 0B
echo [INFO] Services will start automatically on system boot.
echo.
echo.
color 0E
echo How to restart services:
color 0F
echo   Restart server:  sc stop %SERVER_SERVICE_NAME% ^&^& sc start %SERVER_SERVICE_NAME%
echo   Restart panel:   sc stop %PANEL_SERVICE_NAME% ^&^& sc start %PANEL_SERVICE_NAME%
echo   Restart both:    sc stop %SERVER_SERVICE_NAME% ^&^& sc start %SERVER_SERVICE_NAME% ^&^& sc stop %PANEL_SERVICE_NAME% ^&^& sc start %PANEL_SERVICE_NAME%
echo.
color 0B
echo Programmer: Ebrahim Shafiei (EbraSha)
echo Email: Prof.Shafiei@Gmail.com
echo.
color 0F
pause
goto MAIN_MENU

:REMOVE_SERVICE
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0C
echo ║                  REMOVING SERVICES                          ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.

:: Check if services exist
set "SERVICES_FOUND=0"

sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    set "SERVICES_FOUND=1"
)

sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if %errorLevel% equ 0 (
    set "SERVICES_FOUND=1"
)

if !SERVICES_FOUND! equ 0 (
    color 0E
    echo [WARNING] Services are not installed.
    color 0B
    echo [INFO] Nothing to remove.
    echo.
    color 0F
    pause
    goto MAIN_MENU
)

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
echo [COMPLETE] Abdal 4iProto Panel and Server services have been completely removed.
color 0B
echo [INFO] Installation files are still in: %INSTALL_DIR%
echo [INFO] You can manually delete this directory if needed.
echo.
color 0B
echo Programmer: Ebrahim Shafiei (EbraSha)
echo Email: Prof.Shafiei@Gmail.com
echo.
color 0F
pause
goto MAIN_MENU

:CHECK_STATUS
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0F
echo ║                   SERVICE STATUS                            ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.

:: Check server service status
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if %errorLevel% neq 0 (
    color 0C
    echo [STATUS] Server service "%SERVER_SERVICE_NAME%" is NOT INSTALLED
) else (
    for /f "tokens=3 delims=: " %%H in ('sc query "%SERVER_SERVICE_NAME%" ^| findstr "        STATE"') do (
        if /i "%%H"=="RUNNING" (
            color 0A
            echo [STATUS] Server service "%SERVER_SERVICE_NAME%" is RUNNING
        ) else if /i "%%H"=="STOPPED" (
            color 0E
            echo [STATUS] Server service "%SERVER_SERVICE_NAME%" is STOPPED
        ) else (
            color 0B
            echo [STATUS] Server service "%SERVER_SERVICE_NAME%" is %%H
        )
    )
    
    for /f "tokens=3 delims=: " %%H in ('sc qc "%SERVER_SERVICE_NAME%" ^| findstr "        START_TYPE"') do (
        if /i "%%H"=="AUTO_START" (
            color 0A
            echo [STARTUP] Server service is set to AUTOMATIC startup
        ) else if /i "%%H"=="DEMAND_START" (
            color 0E
            echo [STARTUP] Server service is set to MANUAL startup
        ) else (
            color 0B
            echo [STARTUP] Server service startup type: %%H
        )
    )
)
echo.

:: Check panel service status
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if %errorLevel% neq 0 (
    color 0C
    echo [STATUS] Panel service "%PANEL_SERVICE_NAME%" is NOT INSTALLED
) else (
    for /f "tokens=3 delims=: " %%H in ('sc query "%PANEL_SERVICE_NAME%" ^| findstr "        STATE"') do (
        if /i "%%H"=="RUNNING" (
            color 0A
            echo [STATUS] Panel service "%PANEL_SERVICE_NAME%" is RUNNING
        ) else if /i "%%H"=="STOPPED" (
            color 0E
            echo [STATUS] Panel service "%PANEL_SERVICE_NAME%" is STOPPED
        ) else (
            color 0B
            echo [STATUS] Panel service "%PANEL_SERVICE_NAME%" is %%H
        )
    )
    
    for /f "tokens=3 delims=: " %%H in ('sc qc "%PANEL_SERVICE_NAME%" ^| findstr "        START_TYPE"') do (
        if /i "%%H"=="AUTO_START" (
            color 0A
            echo [STARTUP] Panel service is set to AUTOMATIC startup
        ) else if /i "%%H"=="DEMAND_START" (
            color 0E
            echo [STARTUP] Panel service is set to MANUAL startup
        ) else (
            color 0B
            echo [STARTUP] Panel service startup type: %%H
        )
    )
)
echo.
color 0B
echo [INFO] You can manage these services from Services.msc (services.msc)
echo.
color 0F
pause
goto MAIN_MENU

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
echo ║  Abdal 4iProto Panel ^& Server Service Manager              ║
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
