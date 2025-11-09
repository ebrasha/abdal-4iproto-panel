// +build !windows

/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : service_other.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2025-11-05 13:23:16
 * Description  : Non-Windows service stub (for Linux and other OS)
 * -------------------------------------------------------------------
 *
 * "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
 * – Ebrahim Shafiei
 *
 **********************************************************************
 */

package main

import "log"

// isWindowsService always returns false on non-Windows systems
func isWindowsService() bool {
	return false
}

// runWindowsService is a stub for non-Windows systems
func runWindowsService() {
	// This should never be called on non-Windows systems
	// If it is, something went wrong
	log.Fatalf("Windows service functionality is not available on this platform")
}

// installWindowsService, uninstallWindowsService, startWindowsService, stopWindowsService
// are stubs for non-Windows systems
func installWindowsService() {
	log.Fatalf("Windows service installation is not available on this platform")
}

func uninstallWindowsService() {
	log.Fatalf("Windows service uninstallation is not available on this platform")
}

func startWindowsService() {
	log.Fatalf("Windows service start is not available on this platform")
}

func stopWindowsService() {
	log.Fatalf("Windows service stop is not available on this platform")
}

// panelWindowsService is a stub for non-Windows systems
type panelWindowsService struct{}
