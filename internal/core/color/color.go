package color

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// ANSI color codes
const (
	Reset          = "\033[0m"
	BoldStyle      = "\033[1m"
	DimStyle       = "\033[2m"
	UnderlineStyle = "\033[4m"

	// Regular colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	// Bright colors
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

var (
	// NoColor disables color output
	NoColor = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"

	// ForceColor forces color output even if not in a terminal
	ForceColor = os.Getenv("FORCE_COLOR") != ""
)

// init sets up color support based on the environment
func init() {
	// On Windows, enable virtual terminal processing for color support
	if runtime.GOOS == "windows" && !NoColor {
		// Windows 10+ supports ANSI colors with virtual terminal processing
		// This is automatically enabled in newer versions
	}
}

// Enabled returns true if color output is enabled
func Enabled() bool {
	if NoColor {
		return false
	}
	if ForceColor {
		return true
	}
	// Check if stdout is a terminal
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Sprint returns a colored string
func Sprint(color, text string) string {
	if !Enabled() {
		return text
	}
	return color + text + Reset
}

// Sprintf returns a formatted colored string
func Sprintf(color, format string, a ...interface{}) string {
	text := fmt.Sprintf(format, a...)
	if !Enabled() {
		return text
	}
	return color + text + Reset
}

// Print prints colored text
func Print(color, text string) {
	fmt.Print(Sprint(color, text))
}

// Printf prints formatted colored text
func Printf(color, format string, a ...interface{}) {
	fmt.Print(Sprintf(color, format, a...))
}

// Println prints colored text with a newline
func Println(color, text string) {
	fmt.Println(Sprint(color, text))
}

// Provider-specific colors
func AWS(text string) string {
	return Sprint(Yellow, text)
}

func Azure(text string) string {
	return Sprint(Blue, text)
}

func GCP(text string) string {
	return Sprint(BrightBlue, text)
}

func DigitalOcean(text string) string {
	return Sprint(Cyan, text)
}

// Status colors
func Success(text string) string {
	return Sprint(Green, text)
}

func Error(text string) string {
	return Sprint(Red, text)
}

func Warning(text string) string {
	return Sprint(Yellow, text)
}

func Info(text string) string {
	return Sprint(Cyan, text)
}

func Dim(text string) string {
	return Sprint(Gray, text)
}

// Semantic colors for different elements
func Header(text string) string {
	return Sprint(BoldStyle+BrightWhite, text)
}

func Subheader(text string) string {
	return Sprint(BoldStyle+Cyan, text)
}

func Label(text string) string {
	return Sprint(BoldStyle+White, text)
}

func Value(text string) string {
	return Sprint(BrightWhite, text)
}

func Command(text string) string {
	return Sprint(BrightGreen, text)
}

func Flag(text string) string {
	return Sprint(Yellow, text)
}

func Path(text string) string {
	return Sprint(BrightBlue, text)
}

func Count(count int) string {
	color := Green
	if count == 0 {
		color = Gray
	} else if count > 100 {
		color = Yellow
	} else if count > 500 {
		color = Red
	}
	return Sprintf(color, "%d", count)
}

// Severity colors for drift
func Critical(text string) string {
	return Sprint(BrightRed, text)
}

func High(text string) string {
	return Sprint(Red, text)
}

func Medium(text string) string {
	return Sprint(Yellow, text)
}

func Low(text string) string {
	return Sprint(Blue, text)
}

// StripColors removes ANSI color codes from a string
func StripColors(text string) string {
	// Remove all ANSI escape sequences
	for _, code := range []string{
		Reset, BoldStyle, DimStyle, UnderlineStyle,
		Black, Red, Green, Yellow, Blue, Magenta, Cyan, White, Gray,
		BrightRed, BrightGreen, BrightYellow, BrightBlue, BrightMagenta, BrightCyan, BrightWhite,
		BgBlack, BgRed, BgGreen, BgYellow, BgBlue, BgMagenta, BgCyan, BgWhite,
	} {
		text = strings.ReplaceAll(text, code, "")
	}
	return text
}

// Box drawing characters with color
func BoxTop() string {
	return Sprint(Gray, "┌"+strings.Repeat("─", 50)+"┐")
}

func BoxBottom() string {
	return Sprint(Gray, "└"+strings.Repeat("─", 50)+"┘")
}

func BoxLine(text string) string {
	return Sprint(Gray, "│ ") + text + Sprint(Gray, " │")
}

func Divider() string {
	return Sprint(Gray, strings.Repeat("─", 52))
}

func DoubleDivider() string {
	return Sprint(Gray, strings.Repeat("═", 52))
}

// Symbols with color
func CheckMark() string {
	return Sprint(Green, "✓")
}

func CrossMark() string {
	return Sprint(Red, "✗")
}

func Arrow() string {
	return Sprint(Cyan, "→")
}

func Bullet() string {
	return Sprint(Gray, "•")
}

// Progress indicator colors
func Spinner(frame string) string {
	return Sprint(Cyan, frame)
}

func ProgressBar(filled, empty string) string {
	return Sprint(Green, filled) + Sprint(Gray, empty)
}

// Table formatting helpers
func TableHeader(headers ...string) string {
	colored := make([]string, len(headers))
	for i, h := range headers {
		colored[i] = Sprint(BoldStyle+Cyan, h)
	}
	return strings.Join(colored, " ")
}

func TableRow(values ...string) string {
	return strings.Join(values, " ")
}

// Conditional coloring based on boolean
func BoolColor(value bool, trueText, falseText string) string {
	if value {
		return Success(trueText)
	}
	return Error(falseText)
}

func StatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "success", "configured", "active", "running", "healthy":
		return CheckMark()
	case "error", "failed", "not configured", "unhealthy":
		return CrossMark()
	case "warning", "degraded", "partial":
		return Sprint(Yellow, "⚠")
	case "pending", "starting", "stopping":
		return Sprint(Cyan, "⟳")
	case "unknown":
		return Sprint(Gray, "?")
	default:
		return Sprint(Gray, "•")
	}
}

// Format helpers
func Bold(text string) string {
	if !Enabled() {
		return text
	}
	return BoldStyle + text + Reset
}

func Underline(text string) string {
	if !Enabled() {
		return text
	}
	return UnderlineStyle + text + Reset
}

func DimText(text string) string {
	if !Enabled() {
		return text
	}
	return DimStyle + text + Reset
}
