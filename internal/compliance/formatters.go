package compliance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"gopkg.in/yaml.v3"
)

// JSONFormatter formats reports as JSON
type JSONFormatter struct{}

// Format formats the report as JSON
func (f *JSONFormatter) Format(report *ComplianceReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// YAMLFormatter formats reports as YAML
type YAMLFormatter struct{}

// Format formats the report as YAML
func (f *YAMLFormatter) Format(report *ComplianceReport) ([]byte, error) {
	return yaml.Marshal(report)
}

// HTMLFormatter formats reports as HTML
type HTMLFormatter struct{}

// Format formats the report as HTML
func (f *HTMLFormatter) Format(report *ComplianceReport) ([]byte, error) {
	tmpl := template.Must(template.New("report").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"formatScore": func(score float64) string {
			return fmt.Sprintf("%.1f%%", score)
		},
		"statusColor": func(status ControlStatus) string {
			switch status {
			case ControlStatusPassed:
				return "green"
			case ControlStatusFailed:
				return "red"
			case ControlStatusPartial:
				return "orange"
			default:
				return "gray"
			}
		},
		"severityColor": func(severity string) string {
			switch severity {
			case "critical":
				return "#d32f2f"
			case "high":
				return "#f57c00"
			case "medium":
				return "#fbc02d"
			case "low":
				return "#388e3c"
			default:
				return "#757575"
			}
		},
	}).Parse(htmlTemplate))
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, report); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// PDFFormatter formats reports as PDF
type PDFFormatter struct{}

// Format formats the report as PDF (stub - would use a PDF library)
func (f *PDFFormatter) Format(report *ComplianceReport) ([]byte, error) {
	// In production, this would use a PDF generation library
	// For now, return HTML that can be converted to PDF
	htmlFormatter := &HTMLFormatter{}
	return htmlFormatter.Format(report)
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            line-height: 1.6;
            color: #333;
            background-color: #f5f5f5;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px;
            border-radius: 10px;
            margin-bottom: 30px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.1);
        }
        
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        
        .metadata {
            display: flex;
            gap: 30px;
            margin-top: 20px;
        }
        
        .metadata-item {
            display: flex;
            flex-direction: column;
        }
        
        .metadata-label {
            font-size: 0.9em;
            opacity: 0.8;
        }
        
        .metadata-value {
            font-size: 1.1em;
            font-weight: 600;
        }
        
        .summary {
            background: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
        }
        
        .summary h2 {
            color: #667eea;
            margin-bottom: 20px;
        }
        
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-top: 20px;
        }
        
        .summary-card {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
        }
        
        .summary-card-value {
            font-size: 2em;
            font-weight: bold;
            color: #667eea;
        }
        
        .summary-card-label {
            color: #666;
            margin-top: 5px;
        }
        
        .compliance-score {
            background: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
            text-align: center;
        }
        
        .score-circle {
            width: 200px;
            height: 200px;
            margin: 0 auto 20px;
            position: relative;
        }
        
        .score-value {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            font-size: 3em;
            font-weight: bold;
        }
        
        .section {
            background: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
        }
        
        .section h2 {
            color: #667eea;
            margin-bottom: 10px;
        }
        
        .section-description {
            color: #666;
            margin-bottom: 20px;
        }
        
        .control {
            border-left: 4px solid #e0e0e0;
            padding: 20px;
            margin-bottom: 20px;
            background: #f8f9fa;
            border-radius: 0 8px 8px 0;
        }
        
        .control.passed {
            border-left-color: #4caf50;
        }
        
        .control.failed {
            border-left-color: #f44336;
        }
        
        .control.partial {
            border-left-color: #ff9800;
        }
        
        .control-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        
        .control-id {
            font-weight: bold;
            color: #667eea;
        }
        
        .control-status {
            padding: 5px 10px;
            border-radius: 20px;
            font-size: 0.9em;
            font-weight: 600;
            text-transform: uppercase;
        }
        
        .status-passed {
            background: #e8f5e9;
            color: #2e7d32;
        }
        
        .status-failed {
            background: #ffebee;
            color: #c62828;
        }
        
        .status-partial {
            background: #fff3e0;
            color: #e65100;
        }
        
        .finding {
            background: #fff;
            border: 1px solid #e0e0e0;
            padding: 15px;
            margin-top: 15px;
            border-radius: 8px;
        }
        
        .finding-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
        }
        
        .finding-severity {
            padding: 3px 8px;
            border-radius: 4px;
            font-size: 0.85em;
            font-weight: 600;
            text-transform: uppercase;
            color: white;
        }
        
        .severity-critical {
            background: #d32f2f;
        }
        
        .severity-high {
            background: #f57c00;
        }
        
        .severity-medium {
            background: #fbc02d;
        }
        
        .severity-low {
            background: #388e3c;
        }
        
        .finding-remediation {
            margin-top: 10px;
            padding: 10px;
            background: #e3f2fd;
            border-left: 3px solid #2196f3;
            border-radius: 0 4px 4px 0;
        }
        
        .footer {
            text-align: center;
            padding: 30px;
            color: #666;
            font-size: 0.9em;
        }
        
        .signature {
            margin-top: 20px;
            padding: 15px;
            background: #f5f5f5;
            border-radius: 8px;
            font-family: monospace;
        }
        
        @media print {
            body {
                background: white;
            }
            
            .container {
                max-width: 100%;
            }
            
            .section {
                page-break-inside: avoid;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
            <div class="metadata">
                <div class="metadata-item">
                    <span class="metadata-label">Report ID</span>
                    <span class="metadata-value">{{.ID}}</span>
                </div>
                <div class="metadata-item">
                    <span class="metadata-label">Generated</span>
                    <span class="metadata-value">{{formatTime .GeneratedAt}}</span>
                </div>
                <div class="metadata-item">
                    <span class="metadata-label">Period</span>
                    <span class="metadata-value">{{formatTime .Period.Start}} - {{formatTime .Period.End}}</span>
                </div>
                <div class="metadata-item">
                    <span class="metadata-label">Type</span>
                    <span class="metadata-value">{{.Type}}</span>
                </div>
            </div>
        </div>
        
        <div class="summary">
            <h2>Executive Summary</h2>
            <div class="summary-grid">
                <div class="summary-card">
                    <div class="summary-card-value">{{.Summary.TotalControls}}</div>
                    <div class="summary-card-label">Total Controls</div>
                </div>
                <div class="summary-card">
                    <div class="summary-card-value">{{.Summary.PassedControls}}</div>
                    <div class="summary-card-label">Passed</div>
                </div>
                <div class="summary-card">
                    <div class="summary-card-value">{{.Summary.FailedControls}}</div>
                    <div class="summary-card-label">Failed</div>
                </div>
                <div class="summary-card">
                    <div class="summary-card-value">{{formatScore .Summary.ComplianceScore}}</div>
                    <div class="summary-card-label">Compliance Score</div>
                </div>
            </div>
        </div>
        
        <div class="summary">
            <h2>Findings Summary</h2>
            <div class="summary-grid">
                <div class="summary-card">
                    <div class="summary-card-value" style="color: #d32f2f;">{{.Summary.CriticalFindings}}</div>
                    <div class="summary-card-label">Critical</div>
                </div>
                <div class="summary-card">
                    <div class="summary-card-value" style="color: #f57c00;">{{.Summary.HighFindings}}</div>
                    <div class="summary-card-label">High</div>
                </div>
                <div class="summary-card">
                    <div class="summary-card-value" style="color: #fbc02d;">{{.Summary.MediumFindings}}</div>
                    <div class="summary-card-label">Medium</div>
                </div>
                <div class="summary-card">
                    <div class="summary-card-value" style="color: #388e3c;">{{.Summary.LowFindings}}</div>
                    <div class="summary-card-label">Low</div>
                </div>
            </div>
        </div>
        
        {{range .Sections}}
        <div class="section">
            <h2>{{.Title}}</h2>
            {{if .Description}}
            <p class="section-description">{{.Description}}</p>
            {{end}}
            
            {{range .Controls}}
            <div class="control {{.Status}}">
                <div class="control-header">
                    <div>
                        <span class="control-id">{{.ID}}</span> - {{.Title}}
                    </div>
                    <span class="control-status status-{{.Status}}">{{.Status}}</span>
                </div>
                <p>{{.Description}}</p>
                
                {{range .Findings}}
                <div class="finding">
                    <div class="finding-header">
                        <strong>{{.Title}}</strong>
                        <span class="finding-severity severity-{{.Severity}}">{{.Severity}}</span>
                    </div>
                    <p>{{.Description}}</p>
                    {{if .Remediation}}
                    <div class="finding-remediation">
                        <strong>Remediation:</strong> {{.Remediation}}
                    </div>
                    {{end}}
                </div>
                {{end}}
            </div>
            {{end}}
        </div>
        {{end}}
        
        <div class="footer">
            <p>This report was automatically generated by DriftMgr Compliance Reporter</p>
            {{if .Signature}}
            <div class="signature">
                <strong>Digital Signature:</strong> {{.Signature}}
            </div>
            {{end}}
        </div>
    </div>
</body>
</html>
`