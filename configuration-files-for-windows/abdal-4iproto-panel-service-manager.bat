@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM **********************************************************************
REM -------------------------------------------------------------------
REM Project Name : Abdal 4iProto Panel
REM File Name    : abdal-4iproto-service-manager.bat
REM Author       : Ebrahim Shafiei (EbraSha)
REM Email        : Prof.Shafiei@Gmail.com
REM Created On   : 2025-12-23 02:54:39
REM Description  : Comprehensive Windows Service Manager for Abdal 4iProto Panel and Server - Install/Uninstall services with admin privileges
REM -------------------------------------------------------------------
REM
REM "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
REM – Ebrahim Shafiei
REM
REM **********************************************************************

REM Set console color scheme
color 0F

REM Service configuration
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

REM Required files definition
set "PANEL_REQUIRED_FILES=abdal-4iproto-panel.json abdal-4iproto-panel-windows.exe"
set "SERVER_REQUIRED_FILES=abdal-4iproto-server-windows.exe blocked_ips.json id_ed25519 id_ed25519.pub server_config.json users.json"

REM Check for admin privileges
net session >nul 2>&1
if !errorLevel! neq 0 (
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

:MAIN_MENU
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
echo ║        Abdal 4iProto Panel ^& Server SERVICE MANAGER           ║
echo ║                                                              ║
color 0A
echo ║  1. Install Services                                         ║
color 0C
echo ║  2. Remove Services Only                                     ║
color 0E
echo ║  3. Complete Removal (Services + Files)                      ║
color 0D
echo ║  4. Check Service Status                                     ║
color 0F
echo ║  5. Exit                                                     ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.
color 0F
set /p choice=Please select an option (1-5): 

if "%choice%"=="1" goto INSTALL_SERVICE_ROUTINE
if "%choice%"=="2" goto REMOVE_SERVICES_ONLY
if "%choice%"=="3" goto REMOVE_COMPLETE
if "%choice%"=="4" goto CHECK_STATUS
if "%choice%"=="5" goto EXIT
color 0C
echo [ERROR] Invalid choice! Please select 1, 2, 3, 4, or 5.
color 0F
timeout /t 2 >nul
goto MAIN_MENU

:INSTALL_SERVICE_ROUTINE
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0A
echo ║                  INSTALLING SERVICES                         ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.

REM 1. Check Server Service Status
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if !errorLevel! neq 0 (
    set "SERVER_INSTALLED=0"
    goto CHECK_PANEL_SERVICE
)

REM If here, Server IS installed
set "SERVER_INSTALLED=1"
color 0A
echo [INFO] Server service "%SERVER_SERVICE_NAME%" is already installed.
echo [INFO] Only Panel service will be installed.
echo.

:CHECK_PANEL_SERVICE
REM 2. Check Panel Service Status
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if !errorLevel! neq 0 (
    set "PANEL_INSTALLED=0"
    goto CHECK_FILES
)

REM If here, Panel IS installed
set "PANEL_INSTALLED=1"
color 0E
echo [WARNING] Panel service "%PANEL_SERVICE_NAME%" is already installed!
echo.
color 0F
set "confirm="
set /p confirm=Do you want to reinstall Panel service? (yes/no): 
if /i not "!confirm!"=="yes" (
    color 0B
    echo.
    echo [INFO] Installation cancelled by user.
    echo.
    color 0F
    pause
    goto MAIN_MENU
)

echo.
color 0B
echo [INFO] Stopping and removing existing Panel service...
sc stop "%PANEL_SERVICE_NAME%" >nul 2>&1
timeout /t 3 >nul
sc delete "%PANEL_SERVICE_NAME%" >nul 2>&1

if !errorLevel! equ 0 (
    color 0A
    echo [SUCCESS] Existing Panel service removed successfully.
) else (
    color 0C
    echo [ERROR] Failed to remove existing Panel service.
    echo.
    color 0F
    pause
    goto MAIN_MENU
)
echo.

:CHECK_FILES
REM 3. Check Required Files
color 0B
echo [INFO] Checking required files for Panel...
echo.

set "MISSING_FILES=0"
for %%F in (%PANEL_REQUIRED_FILES%) do (
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
    echo [ERROR] Some required Panel files are missing!
    echo.
    color 0E
    echo [INFO] Please make sure all required files are in the current directory:
    echo    !CURRENT_DIR!
    echo.
    echo [INFO] Required Panel files:
    for %%F in (%PANEL_REQUIRED_FILES%) do (
        echo    - %%F
    )
    echo.
    color 0F
    pause
    goto MAIN_MENU
)

echo.
color 0A
echo [SUCCESS] All required Panel files found!
echo.

REM Check Server files ONLY if Server service is NOT installed
if !SERVER_INSTALLED! equ 1 goto INSTALL_EXECUTION

color 0B
echo [INFO] Checking required files for Server...
echo.

set "MISSING_SERVER_FILES=0"
for %%F in (%SERVER_REQUIRED_FILES%) do (
    if not exist "!CURRENT_DIR!\%%F" (
        color 0C
        echo [ERROR] Missing file: %%F
        set "MISSING_SERVER_FILES=1"
    ) else (
        color 0A
        echo [SUCCESS] Found: %%F
    )
)

if !MISSING_SERVER_FILES! equ 1 (
    echo.
    color 0C
    echo [ERROR] Some required Server files are missing!
    echo.
    color 0E
    echo [INFO] Please make sure all required files are in the current directory:
    echo    !CURRENT_DIR!
    echo.
    echo [INFO] Required Server files:
    for %%F in (%SERVER_REQUIRED_FILES%) do (
        echo    - %%F
    )
    echo.
    color 0F
    pause
    goto MAIN_MENU
)

echo.
color 0A
echo [SUCCESS] All required Server files found!
echo.

:INSTALL_EXECUTION
REM 4. Execute Installation
color 0B
echo [INFO] Installation directory: %INSTALL_DIR%
color 0A
echo [SUCCESS] Files are already in the installation directory
echo.

REM Install Server Service if needed
if !SERVER_INSTALLED! equ 1 goto SKIP_SERVER_INSTALL

color 0B
echo [INFO] Installing server service "%SERVER_SERVICE_NAME%"...
set "SERVER_BINPATH=!INSTALL_DIR!\!SERVER_EXECUTABLE! -service"
sc create "%SERVER_SERVICE_NAME%" binPath= "!SERVER_BINPATH!" DisplayName= "%SERVER_SERVICE_DISPLAY_NAME%" start= auto

if !errorLevel! equ 0 (
    color 0A
    echo [SUCCESS] Server service created successfully!
    sc description "%SERVER_SERVICE_NAME%" "%SERVER_SERVICE_DESCRIPTION%"
    
    color 0B
    echo [INFO] Starting server service...
    sc start "%SERVER_SERVICE_NAME%"
    
    if !errorLevel! equ 0 (
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
goto INSTALL_PANEL_SERVICE

:SKIP_SERVER_INSTALL
color 0A
echo [INFO] Server service is already installed. Skipping Server installation.
echo.

:INSTALL_PANEL_SERVICE
REM Install Panel Service
color 0B
echo [INFO] Installing panel service "%PANEL_SERVICE_NAME%"...

REM Try to install Event Log source (requires admin privileges)
color 0B
echo [INFO] Installing Event Log source for "%PANEL_SERVICE_NAME%"...
eventcreate /ID 1 /L APPLICATION /T INFORMATION /SO "%PANEL_SERVICE_NAME%" /D "Abdal 4iProto Panel Event Log Source" >nul 2>&1
if !errorLevel! equ 0 (
    color 0A
    echo [SUCCESS] Event Log source installed successfully!
) else (
    REM Try PowerShell method
    powershell -Command "New-EventLog -LogName Application -Source '%PANEL_SERVICE_NAME%' -ErrorAction SilentlyContinue" >nul 2>&1
    if !errorLevel! equ 0 (
        color 0A
        echo [SUCCESS] Event Log source installed successfully via PowerShell!
    ) else (
        color 0E
        echo [WARNING] Could not install Event Log source automatically.
        echo [INFO] Event Log will be created automatically when the service starts.
        echo [INFO] You can install it manually later if needed.
    )
)
echo.

set "PANEL_BINPATH=!INSTALL_DIR!\!PANEL_EXECUTABLE!"
sc create "%PANEL_SERVICE_NAME%" binPath= "!PANEL_BINPATH!" DisplayName= "%PANEL_SERVICE_DISPLAY_NAME%" start= auto

if !errorLevel! equ 0 (
    color 0A
    echo [SUCCESS] Panel service created successfully!
    sc description "%PANEL_SERVICE_NAME%" "%PANEL_SERVICE_DESCRIPTION%"
    
    color 0B
    echo [INFO] Starting panel service...
    sc start "%PANEL_SERVICE_NAME%"
    
    if !errorLevel! equ 0 (
        color 0A
        echo [SUCCESS] Panel service started successfully!
    ) else (
        color 0E
        echo [WARNING] Panel service created but failed to start.
        color 0B
        echo [INFO] You can start it manually from Services.msc
        echo [INFO] Check Event Viewer for error details.
    )
) else (
    color 0C
    echo [ERROR] Failed to create panel service!
    color 0E
    echo [INFO] Please check the executable path and permissions.
)
echo.

color 0A
if !SERVER_INSTALLED! equ 0 (
    echo [COMPLETE] Abdal 4iProto Panel and Server are now running as Windows services.
) else (
    echo [COMPLETE] Abdal 4iProto Panel is now running as a Windows service.
    echo [INFO] Server service was already installed and was not modified.
)
color 0B
echo [INFO] Services will start automatically on system boot.
echo.
echo.
color 0E
echo How to restart services:
color 0F
echo    Restart server:  sc stop %SERVER_SERVICE_NAME% ^&^& sc start %SERVER_SERVICE_NAME%
echo    Restart panel:   sc stop %PANEL_SERVICE_NAME% ^&^& sc start %PANEL_SERVICE_NAME%
echo    Restart both:    sc stop %SERVER_SERVICE_NAME% ^&^& sc start %SERVER_SERVICE_NAME% ^&^& sc stop %PANEL_SERVICE_NAME% ^&^& sc start %PANEL_SERVICE_NAME%
echo.
color 0B
echo Programmer: Ebrahim Shafiei (EbraSha)
echo Email: Prof.Shafiei@Gmail.com
echo.
color 0F
pause
goto MAIN_MENU

:REMOVE_SERVICES_ONLY
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0E
echo ║               REMOVE SERVICES ONLY                           ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.
color 0E
echo [WARNING] This will remove the following services:
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
    echo    • %PANEL_SERVICE_NAME%
)
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
    echo    • %SERVER_SERVICE_NAME%
)
echo.
color 0E
echo [WARNING] Files will remain in the installation directory.
echo.
color 0F
set "confirm="
set /p confirm=Are you sure you want to continue? (yes/no): 

if /i not "!confirm!"=="yes" (
    color 0B
    echo.
    echo [INFO] Operation cancelled by user.
    echo.
    color 0F
    pause
    goto MAIN_MENU
)

echo.
color 0B
echo [INFO] Starting service removal...
echo.

REM Check if services exist
set "SERVICES_FOUND=0"

sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
    set "SERVICES_FOUND=1"
)

sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
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

REM Stop and remove server service
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
    color 0B
    echo [INFO] Stopping server service "%SERVER_SERVICE_NAME%"...
    sc stop "%SERVER_SERVICE_NAME%" >nul 2>&1
    
    if !errorLevel! equ 0 (
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
    
    if !errorLevel! equ 0 (
        color 0A
        echo [SUCCESS] Server service removed successfully!
    ) else (
        color 0C
        echo [ERROR] Failed to remove server service!
    )
    echo.
)

REM Stop and remove panel service
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
    color 0B
    echo [INFO] Stopping panel service "%PANEL_SERVICE_NAME%"...
    sc stop "%PANEL_SERVICE_NAME%" >nul 2>&1
    
    if !errorLevel! equ 0 (
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
    
    if !errorLevel! equ 0 (
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
goto MAIN_MENU

:REMOVE_COMPLETE
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0C
echo ║               COMPLETE REMOVAL                               ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.
color 0C
echo [WARNING] This will remove the following services:
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
    echo    • %PANEL_SERVICE_NAME%
)
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
    echo    • %SERVER_SERVICE_NAME%
)
echo.
color 0C
echo [WARNING] This will also DELETE ALL FILES in the installation directory:
echo    %INSTALL_DIR%
echo.
color 0C
echo [WARNING] The following files will be deleted:
set "ALL_FILES=%PANEL_REQUIRED_FILES% %SERVER_REQUIRED_FILES%"
for %%F in (%ALL_FILES%) do (
    if exist "!INSTALL_DIR!\%%F" (
        echo    - %%F
    )
)
echo.
color 0C
echo [WARNING] This action cannot be undone!
echo.
color 0F
set "confirm="
set /p confirm=Are you absolutely sure you want to continue? (yes/no): 

if /i not "!confirm!"=="yes" (
    color 0B
    echo.
    echo [INFO] Operation cancelled by user.
    echo.
    color 0F
    pause
    goto MAIN_MENU
)

echo.
color 0B
echo [INFO] Starting complete removal...
echo.

REM Stop and remove server service
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
    color 0B
    echo [INFO] Stopping server service "%SERVER_SERVICE_NAME%"...
    sc stop "%SERVER_SERVICE_NAME%" >nul 2>&1
    
    if !errorLevel! equ 0 (
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
    
    if !errorLevel! equ 0 (
        color 0A
        echo [SUCCESS] Server service removed successfully!
    ) else (
        color 0C
        echo [ERROR] Failed to remove server service!
    )
    echo.
)

REM Stop and remove panel service
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if !errorLevel! equ 0 (
    color 0B
    echo [INFO] Stopping panel service "%PANEL_SERVICE_NAME%"...
    sc stop "%PANEL_SERVICE_NAME%" >nul 2>&1
    
    if !errorLevel! equ 0 (
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
    
    if !errorLevel! equ 0 (
        color 0A
        echo [SUCCESS] Panel service removed successfully!
    ) else (
        color 0C
        echo [ERROR] Failed to remove panel service!
    )
    echo.
)

REM Delete files
color 0B
echo [INFO] Deleting files from installation directory...
echo.

set "FILES_DELETED=0"
set "ALL_FILES=%PANEL_REQUIRED_FILES% %SERVER_REQUIRED_FILES%"
for %%F in (%ALL_FILES%) do (
    if exist "!INSTALL_DIR!\%%F" (
        del /F /Q "!INSTALL_DIR!\%%F" >nul 2>&1
        if !errorLevel! equ 0 (
            color 0A
            echo [SUCCESS] Deleted: %%F
            set "FILES_DELETED=1"
        ) else (
            color 0C
            echo [ERROR] Failed to delete: %%F
        )
    )
)

REM Try to remove directory if empty
if exist "%INSTALL_DIR%" (
    rmdir "%INSTALL_DIR%" >nul 2>&1
    if !errorLevel! equ 0 (
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
goto MAIN_MENU

:CHECK_STATUS
cls
color 0B
echo.
echo ╔══════════════════════════════════════════════════════════════╗
color 0F
echo ║                   SERVICE STATUS                             ║
color 0B
echo ╚══════════════════════════════════════════════════════════════╝
echo.

REM Check server service status
sc query "%SERVER_SERVICE_NAME%" >nul 2>&1
if !errorLevel! neq 0 (
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

REM Check panel service status
sc query "%PANEL_SERVICE_NAME%" >nul 2>&1
if !errorLevel! neq 0 (
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
echo ║                      THANK YOU!                              ║
color 0B
echo ║                                                              ║
color 0F
echo ║  Abdal 4iProto Panel ^& Server Service Manager               ║
echo ║  Developed by: Ebrahim Shafiei (EbraSha)                   ║
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