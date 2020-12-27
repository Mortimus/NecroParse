package main

import (
	"log"
	"time"
)

// Logs is for logging, with extended functionality
type Logs struct {
	funcName  string
	startTime time.Time
	errors    int
	warnings  int
	infos     int
	debugs    int
}

// Init should be added at the start of every function
func (l *Logs) Init(name string) {
	l.funcName = name
	l.startTime = time.Now()
}

// LogInit is a quicker way to init logs
func LogInit(name string) *Logs {
	l := Logs{}
	l.Init(name)
	return &l
}

// End should be deferred directly after init
func (l *Logs) End() {
	// endTime := time.Now()
	elapsed := time.Since(l.startTime)
	l.TraceF("%s took %+v", l.funcName, elapsed)
}

// 0=Off,1=Error,2=Warn,3=Info,4=Debug,5=Trace

// TraceF is for logging analytical data
func (l *Logs) TraceF(format string, v ...interface{}) {
	if configuration.LogLevel >= 5 {
		log.Printf("[TRACE]["+l.funcName+"] "+format, v...)
	}
}

// DebugF is for printing debug info
func (l *Logs) DebugF(format string, v ...interface{}) {
	l.debugs++
	if configuration.LogLevel >= 4 {
		log.Printf("[DEBUG]["+l.funcName+"] "+format, v...)
	}
}

// InfoF is for printing informationals
func (l *Logs) InfoF(format string, v ...interface{}) {
	l.infos++
	if configuration.LogLevel >= 3 {
		log.Printf("[INFO]["+l.funcName+"] "+format, v...)
	}
}

// WarnF is for printing warnings
func (l *Logs) WarnF(format string, v ...interface{}) {
	l.warnings++
	if configuration.LogLevel >= 2 {
		log.Printf("[WARN]["+l.funcName+"] "+format, v...)
	}
}

// ErrorF is for printing errors
func (l *Logs) ErrorF(format string, v ...interface{}) {
	l.errors++
	if configuration.LogLevel >= 1 {
		log.Printf("[ERROR]["+l.funcName+"] "+format, v...)
	}
}

// FatalF is for printing fatal errors
func (l *Logs) FatalF(format string, v ...interface{}) {
	l.errors++
	log.Fatalf("[ERROR]["+l.funcName+"] "+format, v...)
}
