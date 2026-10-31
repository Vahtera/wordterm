//go:build windows

package storage

import "os"

func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid) // fails if the process does not exist
	if err != nil {
		return false
	}
	p.Release()
	return true
}
