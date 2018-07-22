package plugin

import "fmt"

const (
	// OK signifies that the plugin was able to check the service and it appeared
	// to be functioning properly
	OK = iota

	// Warning signifinies that the plugin was able to check the service, but it
	// appeared to be above some "warning" threshold or did not appear to be
	// working properly
	Warning

	// Critical signifies that the plugin detected that either the service was not
	// running or it was above some "critical" threshold
	Critical

	// Unknown signifies that invalid command line arguments were supplied to the
	// plugin or low-level failures internal to the plugin (such as unable to
	// fork, or open a tcp socket) that prevent it from performing the specified
	// operation. Higher-level errors (such as name resolution errors, socket
	// timeouts, etc) are outside of the control of plugins and should generally
	// NOT be reported as UNKNOWN states.
	Unknown
)

// Statuses represents the human-readable statuses
var Statuses = []string{"OK", "WARNING", "CRITICAL", "UNKNOWN"}

// Exit is a custom error type that contains a return code and a message
type Exit struct {
	Msg    string
	Status int
}

// Error returns the string representation of a return code
func (e *Exit) Error() string {
	return fmt.Sprintf("%s: %s\n", Statuses[e.Status], e.Msg)
}
