package log

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

var (
	useStackDriverFormat = os.Getenv("GAE_SERVICE") != ""
	projectID            = os.Getenv("GOOGLE_CLOUD_PROJECT")
	requestCounter       int64
)

type traceKey int

func ContextWithTraceID(r *http.Request) context.Context {
	var trace string
	if useStackDriverFormat {
		h := r.Header.Get("X-Cloud-Trace-Context")
		if i := strings.IndexByte(h, '/'); i > 0 {
			trace = fmt.Sprintf("projects/%s/traces/%s", projectID, h[:i])
		}
	} else {
		trace = fmt.Sprintf("%08d: ", atomic.AddInt64(&requestCounter, 1))
	}
	return context.WithValue(r.Context(), traceKey(0), trace)
}

type Severity string

const (
	Debug     Severity = "DEBUG"     // Debug or trace information.
	Info      Severity = "INFO"      // Routine information, such as ongoing status or performance.
	Notice    Severity = "NOTICE"    // Normal but significant events, such as start up, shut down, or a configuration change.
	Warning   Severity = "WARNING"   // Warning events might cause problems.
	Error     Severity = "ERROR"     // Error events are likely to cause problems.
	Critical  Severity = "CRITICAL"  // Critical events cause more severe problems or outages.
	Alert     Severity = "ALERT"     // A person must take an action immediately.
	Emergency Severity = "EMERGENCY" // One or more systems are unusable.
)

// Logf logs the given message.
//
// If running on App Engine, the message is logged in a JSON format understood
// by StackDriver.
func Logf(ctx context.Context, severity Severity, format string, args ...interface{}) {
	trace, _ := ctx.Value(traceKey(0)).(string)
	if useStackDriverFormat {
		if severity == "" {
			severity = Info
		}
		entry := struct {
			Message  string `json:"message"`
			Severity string `json:"severity,omitempty"`
			Trace    string `json:"logging.googleapis.com/trace,omitempty"`
		}{
			Message:  fmt.Sprintf(format, args...),
			Severity: string(severity),
			Trace:    trace,
		}
		p, _ := json.Marshal(&entry)
		p = append(p, '\n')
		os.Stderr.Write(p)
	} else {
		log.Printf(trace+format, args...)
	}
}
