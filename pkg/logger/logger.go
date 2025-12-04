package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

var (
	debugMode bool
	mu        sync.RWMutex
	output    io.Writer = os.Stderr
)

// SetDebug enables or disables debug logging
func SetDebug(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	debugMode = enabled
	if enabled {
		log.SetOutput(output)
	} else {
		log.SetOutput(io.Discard)
	}
}

// SetOutput sets the output writer for debug logs
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	output = w
	if debugMode {
		log.SetOutput(w)
	}
}

// IsDebug returns whether debug mode is enabled
func IsDebug() bool {
	mu.RLock()
	defer mu.RUnlock()
	return debugMode
}

// Debug logs a message only if debug mode is enabled
func Debug(format string, v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	if debugMode {
		log.Printf(format, v...)
	}
}

// Debugln logs a message with newline only if debug mode is enabled
func Debugln(v ...interface{}) {
	mu.RLock()
	defer mu.RUnlock()
	if debugMode {
		log.Println(v...)
	}
}

// Error logs error messages (always printed, even in silent mode)
func Error(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", v...)
}

// Fatal logs error and exits (always printed)
func Fatal(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

// Fatalf logs formatted error and exits (always printed)
func Fatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", v...)
	os.Exit(1)
}
