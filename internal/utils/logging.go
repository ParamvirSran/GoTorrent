package utils

import "log"

// DebugLog only prints when DEBUG=1 is passed as env variable
func DebugLog(debugMode bool, format string, v ...interface{}) {
	if debugMode {
		log.Printf("[DEBUG] "+format, v...)
	}
}
