//go:build windows
// +build windows

/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : service_windows.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2025-11-05 13:23:16
 * Description  : Windows service implementation for Abdal 4iProto Panel
 * -------------------------------------------------------------------
 *
 * "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
 * – Ebrahim Shafiei
 *
 **********************************************************************
 */

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	svcmgr "golang.org/x/sys/windows/svc/mgr"
)

// isWindowsService checks if the program is running as a Windows service
func isWindowsService() bool {
	// Check command line arguments - if installing/uninstalling service, handle separately
	if len(os.Args) > 1 {
		arg := strings.ToLower(os.Args[1])
		if arg == "install" || arg == "uninstall" || arg == "start" || arg == "stop" {
			return false // These are service management commands, not service execution
		}
	}

	// Use svc.IsAnInteractiveSession() to detect if running as service
	// Services run in non-interactive sessions
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		return false
	}
	return !isIntSess // If not interactive, it's a service
}

// runWindowsService runs the application as a Windows service
func runWindowsService() {
	// Check command line arguments for service management
	if len(os.Args) > 1 {
		arg := strings.ToLower(os.Args[1])
		switch arg {
		case "install":
			installWindowsService()
			return
		case "uninstall":
			uninstallWindowsService()
			return
		case "start":
			startWindowsService()
			return
		case "stop":
			stopWindowsService()
			return
		}
	}

	// Run as service
	panelLogger = NewLogger(true) // Always log when running as service
	panelLogger.Info("Running as Windows Service...")

	// Use svc.Run for production service
	// For testing/debugging, you can use debug.Run instead
	useDebug := os.Getenv("SERVICE_DEBUG") == "1"
	var err error
	if useDebug {
		err = debug.Run(serviceName, &panelWindowsService{})
	} else {
		err = svc.Run(serviceName, &panelWindowsService{})
	}
	if err != nil {
		panelLogger.Error("Windows service failed", err)
		log.Fatalf("Windows service failed: %v", err)
	}
}

// panelWindowsService implements the svc.Handler interface for Windows services
type panelWindowsService struct{}

// Execute implements the svc.Handler interface
func (ws *panelWindowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	// Set working directory to executable directory to ensure config files are found
	// This is critical because Windows services may run from System32 by default
	exePath, err := os.Executable()
	if err == nil {
		exePath, err = filepath.Abs(exePath)
		if err == nil {
			exeDir := filepath.Dir(exePath)
			if err := os.Chdir(exeDir); err == nil {
				// panelLogger may not be initialized yet, use log instead
				log.Printf("Working directory set to: %s", exeDir)
			} else {
				log.Printf("Failed to set working directory: %v", err)
			}
		}
	}

	// Start server in a goroutine
	go func() {
		runServer()
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	// Wait for service control commands
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				// Stop the server
				if httpServer != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					httpServer.Shutdown(ctx)
					cancel()
				}
				cleanup()
				return
			default:
				panelLogger.Warning(fmt.Sprintf("Unexpected service control request: %d", c.Cmd))
			}
		case <-serverStop:
			changes <- svc.Status{State: svc.StopPending}
			cleanup()
			return
		}
	}
}

// installWindowsService installs the application as a Windows service
func installWindowsService() {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	exePath, err = filepath.Abs(exePath)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	// Get the directory where the executable is located (this will be the working directory)
	exeDir := filepath.Dir(exePath)

	m, err := svcmgr.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		log.Fatalf("Service %s already exists", serviceName)
	}

	config := svcmgr.Config{
		DisplayName:  "Abdal 4iProto Panel",
		Description:  "Management panel for Abdal 4iProto Server",
		StartType:    svcmgr.StartAutomatic,
		ErrorControl: svcmgr.ErrorNormal,
	}

	s, err = m.CreateService(serviceName, exePath, config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}
	defer s.Close()

	// Note: Windows Service Manager doesn't support setting WorkingDirectory in Config
	// The working directory will be set programmatically in Execute() and runServer()
	// to ensure config files are found even if service runs from System32

	fmt.Printf("Service %s installed successfully\n", serviceName)
	fmt.Printf("Executable directory: %s\n", exeDir)
	fmt.Printf("Note: Working directory will be set to executable directory at runtime\n")
	fmt.Printf("To start the service, run: sc start %s\n", serviceName)
}

// uninstallWindowsService uninstalls the Windows service
func uninstallWindowsService() {
	m, err := svcmgr.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		log.Fatalf("Service %s is not installed", serviceName)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		log.Fatalf("Failed to delete service: %v", err)
	}

	fmt.Printf("Service %s uninstalled successfully\n", serviceName)
}

// startWindowsService starts the Windows service
func startWindowsService() {
	m, err := svcmgr.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		log.Fatalf("Service %s is not installed", serviceName)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}

	fmt.Printf("Service %s started successfully\n", serviceName)
}

// stopWindowsService stops the Windows service
func stopWindowsService() {
	m, err := svcmgr.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		log.Fatalf("Service %s is not installed", serviceName)
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	if err != nil {
		log.Fatalf("Failed to stop service: %v", err)
	}

	fmt.Printf("Service %s stopped successfully\n", serviceName)
}
