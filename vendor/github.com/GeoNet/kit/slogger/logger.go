package slogger

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

/*
SmartLogger wraps around the standard logger and adds functionality to avoid repeated logs.
usage:
logger := NewSmartLogger(2*time.Second, "error connecting to:")
logger.Log("error connecting to AVLN")
logger.Log("error connecting to AUCK")
*/
type SmartLogger struct {
	window         time.Duration //time window to calculate repeated messages
	repeatedPrefix string        //message prefix to evaluate, compare whole message if not specified

	mu          sync.Mutex
	lastMessage string
	lastLogTime time.Time
	repeatCount int
}

// NewSmartLogger creates a new SmartLogger with the given time window for detecting repeated messages.
// and an optional predefined message prefix
func NewSmartLogger(window time.Duration, repeatedPrefix string) *SmartLogger {
	sl := &SmartLogger{
		window:         window,
		repeatedPrefix: repeatedPrefix,
	}
	return sl
}

// Log logs a message, checking if it is repeated within the time window
// return the repeatCount
func (sl *SmartLogger) Log(message ...any) int {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	now := time.Now()
	msgString := fmt.Sprintln(message...)
	repeated := sl.checkRepeated(msgString)
	if repeated && now.Sub(sl.lastLogTime) <= sl.window {
		sl.repeatCount++
	} else {
		if sl.repeatedPrefix != "" && !repeated { //this is a random message
			log.Println(msgString)
			sl.lastMessage = ""          // Reset lastMessage to avoid tracking it as a repeated message
			sl.lastLogTime = time.Time{} // Reset the time
			return 0
		}
		sl.flush()
		sl.lastMessage = msgString
		sl.lastLogTime = now
		sl.repeatCount = 1
	}
	return sl.repeatCount
}

// flush writes out the summary of repeated messages
func (sl *SmartLogger) flush() {
	if sl.repeatCount > 1 {
		if sl.repeatedPrefix != "" {
			log.Printf("message with prefix \"%s\" repeated %d times", sl.repeatedPrefix, sl.repeatCount)
		} else {
			log.Printf("message \"%s\" repeated %d times", sl.lastMessage, sl.repeatCount)
		}
		sl.repeatCount = 0
	}
}

// checks if message is repeated
func (sl *SmartLogger) checkRepeated(message string) bool {
	if sl.repeatedPrefix != "" {
		return strings.HasPrefix(message, sl.repeatedPrefix)
	} else {
		return message == sl.lastMessage
	}
}
