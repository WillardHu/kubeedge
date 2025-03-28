package printer

import "context"

type StepDone func()

type Printer interface {
	// SetDebug set debug mode.
	// if debug is true, all debug messages and error stack will be printed.
	SetDebug(debug bool)

	// Debug print debug message.
	Debug(msg string)

	// Debugf print debug message with format.
	Debugf(format string, args ...any)

	// Info print info message.
	Info(msg string)

	// Infof print info message with format.
	Infof(format string, args ...any)

	// Warn print warning message.
	Warn(msg string)

	// Warnf print warning message with format.
	Warnf(format string, args ...any)

	// Error print error message.
	Error(err error, msg string)

	// Errorf print error message with format.
	Errorf(err error, format string, args ...any)

	// Step print step message.
	// StepDone is a function that can be called to print the step done message.
	Step(msg string) StepDone

	// Stepf print step message with format.
	// StepDone is a function that can be called to print the step done message.
	Stepf(format string, args ...any) StepDone

	// StepAndRun print step message and run the function.
	StepAndRun(fn func() error, msg string) error

	// StepfAndRun print step message with format and run the function.
	StepfAndRun(fn func() error, format string, args ...any) error

	// Input print input message and return user input.
	Input(msg string) (string, error)

	// Inputf print input message with format and return user input.
	Inputf(format string, args ...any) (string, error)
}

type printerContextKey struct{}

// FromContext returns printer from context.
func FromContext(ctx context.Context) Printer {
	p, ok := ctx.Value(printerContextKey{}).(Printer)
	if !ok {
		return nil
	}
	return p
}

// NewContext creates a new context with printer.
func NewContext(ctx context.Context, p Printer) context.Context {
	return context.WithValue(ctx, printerContextKey{}, p)
}
