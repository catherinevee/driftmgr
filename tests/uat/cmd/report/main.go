package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type TestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Output  string    `json:"Output"`
	Elapsed float64   `json:"Elapsed"`
}

func main() {
	var (
		inputPath  = flag.String("input", "", "Path to test JSON output")
		persona    = flag.String("persona", "", "Persona name")
		outputPath = flag.String("output", "", "Path to output HTML report")
	)
	flag.Parse()

	if *inputPath == "" {
		log.Fatal("Input path is required")
	}
	if *outputPath == "" {
		log.Fatal("Output path is required")
	}

	// Parse test output
	file, err := os.Open(*inputPath)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer file.Close()

	var events []TestEvent
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var event TestEvent
		if err := decoder.Decode(&event); err != nil {
			continue // Skip malformed lines
		}
		events = append(events, event)
	}

	// Generate HTML report
	html := generateHTMLReport(events, *persona)

	// Write output
	if err := os.WriteFile(*outputPath, []byte(html), 0644); err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}

	fmt.Printf("UAT report generated: %s\n", *outputPath)

	// Print summary
	passed, failed := countResults(events)
	fmt.Printf("Results: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1) // Exit with error if tests failed
	}
}

func countResults(events []TestEvent) (passed, failed int) {
	testResults := make(map[string]string)

	for _, event := range events {
		if event.Test != "" {
			switch event.Action {
			case "pass":
				testResults[event.Test] = "pass"
			case "fail":
				testResults[event.Test] = "fail"
			}
		}
	}

	for _, result := range testResults {
		if result == "pass" {
			passed++
		} else {
			failed++
		}
	}

	return passed, failed
}

func generateHTMLReport(events []TestEvent, persona string) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
    <title>UAT Report - ` + persona + `</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #007bff;
            padding-bottom: 10px;
        }
        .summary {
            display: flex;
            gap: 20px;
            margin: 20px 0;
        }
        .stat {
            flex: 1;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 4px;
            text-align: center;
        }
        .stat.pass {
            background: #d4edda;
            color: #155724;
        }
        .stat.fail {
            background: #f8d7da;
            color: #721c24;
        }
        .test {
            margin: 10px 0;
            padding: 10px;
            border-left: 3px solid #dee2e6;
        }
        .test.pass {
            border-color: #28a745;
        }
        .test.fail {
            border-color: #dc3545;
        }
        .output {
            margin-top: 5px;
            padding: 10px;
            background: #f8f9fa;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
            font-size: 12px;
            white-space: pre-wrap;
        }
        .timestamp {
            color: #6c757d;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>UAT Report: ` + formatPersona(persona) + `</h1>
        <p class="timestamp">Generated: ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
`)

	// Add summary
	passed, failed := countResults(events)
	total := passed + failed
	passRate := 0.0
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}

	sb.WriteString(`
        <div class="summary">
            <div class="stat">
                <h3>Total Tests</h3>
                <p style="font-size: 24px; font-weight: bold;">` + fmt.Sprintf("%d", total) + `</p>
            </div>
            <div class="stat pass">
                <h3>Passed</h3>
                <p style="font-size: 24px; font-weight: bold;">` + fmt.Sprintf("%d", passed) + `</p>
            </div>
            <div class="stat fail">
                <h3>Failed</h3>
                <p style="font-size: 24px; font-weight: bold;">` + fmt.Sprintf("%d", failed) + `</p>
            </div>
            <div class="stat">
                <h3>Pass Rate</h3>
                <p style="font-size: 24px; font-weight: bold;">` + fmt.Sprintf("%.1f%%", passRate) + `</p>
            </div>
        </div>

        <h2>Test Results</h2>
`)

	// Group tests
	tests := make(map[string][]TestEvent)
	for _, event := range events {
		if event.Test != "" {
			tests[event.Test] = append(tests[event.Test], event)
		}
	}

	// Display each test
	for testName, testEvents := range tests {
		status := "pending"
		var output strings.Builder
		var elapsed float64

		for _, event := range testEvents {
			if event.Action == "pass" || event.Action == "fail" {
				status = event.Action
				elapsed = event.Elapsed
			}
			if event.Output != "" && !strings.HasPrefix(event.Output, "===") {
				output.WriteString(event.Output)
			}
		}

		sb.WriteString(`<div class="test ` + status + `">`)
		sb.WriteString(`<h3>` + testName + ` <span style="float: right; font-size: 14px;">`)
		if status == "pass" {
			sb.WriteString(`✅ PASS`)
		} else if status == "fail" {
			sb.WriteString(`❌ FAIL`)
		}
		if elapsed > 0 {
			sb.WriteString(fmt.Sprintf(` (%.2fs)`, elapsed))
		}
		sb.WriteString(`</span></h3>`)

		if output.Len() > 0 {
			sb.WriteString(`<div class="output">` + strings.TrimSpace(output.String()) + `</div>`)
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`
    </div>
</body>
</html>`)

	return sb.String()
}

func formatPersona(persona string) string {
	// Format persona name for display
	switch persona {
	case "devops_engineer":
		return "DevOps Engineer"
	case "platform_engineer":
		return "Platform Engineer"
	case "sre":
		return "Site Reliability Engineer"
	case "security_engineer":
		return "Security Engineer"
	default:
		return strings.Title(strings.ReplaceAll(persona, "_", " "))
	}
}
