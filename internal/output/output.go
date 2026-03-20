package output

import (
	"fmt"
	"os"
)

// Colour for terminal output
type Colour string

const (
	ColourReset   Colour = "\033[0m"
	ColourBold    Colour = "\033[1m"
	ColourRed     Colour = "\033[31m"
	ColourGreen   Colour = "\033[32m"
	ColourYellow  Colour = "\033[33m"
	ColourBlue    Colour = "\033[34m"
	ColourMagenta Colour = "\033[35m"
	ColourCyan    Colour = "\033[36m"
)

func Line(msg string) {
	fmt.Fprintln(os.Stdout, msg)
}

func Linef(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

func LineColour(msg string, c Colour) {
	fmt.Fprint(os.Stdout, string(c), msg, string(ColourReset), "\n")
}

func LineColourf(c Colour, format string, args ...any) {
	fmt.Fprint(os.Stdout, string(c))
	fmt.Fprintf(os.Stdout, format+"\n", args...)
	fmt.Fprint(os.Stdout, string(ColourReset))
}

func Info(msg string) {
	LineColour(msg, ColourCyan)
}

func Infof(format string, args ...any) {
	LineColourf(ColourCyan, format, args...)
}

func Success(msg string) {
	LineColour(msg, ColourGreen)
}

func Successf(format string, args ...any) {
	LineColourf(ColourGreen, format, args...)
}

func Error(msg string) {
	LineColour(msg, ColourRed)
}

func Errorf(format string, args ...any) {
	LineColourf(ColourRed, format, args...)
}

func Warn(msg string) {
	LineColour(msg, ColourYellow)
}
