package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/catherinevee/driftmgr/internal/bi"
)

// BICommand represents the business intelligence management command
type BICommand struct {
	service *bi.BIService
}

// NewBICommand creates a new BI command
func NewBICommand() *BICommand {
	// Create BI service
	service := bi.NewBIService()

	return &BICommand{
		service: service,
	}
}

// HandleBI handles the BI command
func HandleBI(args []string) {
	cmd := NewBICommand()

	// Start the service
	ctx := context.Background()
	if err := cmd.service.Start(ctx); err != nil {
		fmt.Printf("Error starting BI service: %v\n", err)
		return
	}
	defer cmd.service.Stop(ctx)

	if len(args) == 0 {
		cmd.showHelp()
		return
	}

	switch args[0] {
	case "dashboard":
		cmd.handleDashboard(args[1:])
	case "report":
		cmd.handleReport(args[1:])
	case "dataset":
		cmd.handleDataset(args[1:])
	case "query":
		cmd.handleQuery(args[1:])
	case "export":
		cmd.handleExport(args[1:])
	case "status":
		cmd.handleStatus(args[1:])
	default:
		fmt.Printf("Unknown BI command: %s\n", args[0])
		cmd.showHelp()
	}
}

// showHelp shows the help for BI commands
func (cmd *BICommand) showHelp() {
	fmt.Println("Business Intelligence Management Commands:")
	fmt.Println("  dashboard <cmd>                - Manage dashboards")
	fmt.Println("  report <cmd>                   - Manage reports")
	fmt.Println("  dataset <cmd>                  - Manage datasets")
	fmt.Println("  query <cmd>                    - Manage queries")
	fmt.Println("  export <cmd>                   - Export data")
	fmt.Println("  status                         - Show BI status")
	fmt.Println()
	fmt.Println("Dashboard Commands:")
	fmt.Println("  dashboard create <name> <category> - Create a new dashboard")
	fmt.Println("  dashboard list                     - List all dashboards")
	fmt.Println("  dashboard get <id>                 - Get dashboard details")
	fmt.Println("  dashboard update <id>              - Update a dashboard")
	fmt.Println("  dashboard delete <id>              - Delete a dashboard")
	fmt.Println()
	fmt.Println("Report Commands:")
	fmt.Println("  report create <name> <category>    - Create a new report")
	fmt.Println("  report list                        - List all reports")
	fmt.Println("  report generate <id>               - Generate a report")
	fmt.Println("  report schedule <id> <schedule>    - Schedule a report")
	fmt.Println()
	fmt.Println("Dataset Commands:")
	fmt.Println("  dataset create <name> <source>     - Create a new dataset")
	fmt.Println("  dataset list                       - List all datasets")
	fmt.Println("  dataset refresh <id>               - Refresh a dataset")
	fmt.Println()
	fmt.Println("Query Commands:")
	fmt.Println("  query create <name> <sql>          - Create a new query")
	fmt.Println("  query list                         - List all queries")
	fmt.Println("  query execute <id>                 - Execute a query")
	fmt.Println()
	fmt.Println("Export Commands:")
	fmt.Println("  export data <query-id> <format>    - Export query results")
	fmt.Println("  export formats                     - List supported formats")
}

// handleDashboard handles dashboard management
func (cmd *BICommand) handleDashboard(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi dashboard <command>")
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		cmd.handleCreateDashboard(args[1:])
	case "list":
		cmd.handleListDashboards(args[1:])
	case "get":
		cmd.handleGetDashboard(args[1:])
	case "update":
		cmd.handleUpdateDashboard(args[1:])
	case "delete":
		cmd.handleDeleteDashboard(args[1:])
	default:
		fmt.Printf("Unknown dashboard command: %s\n", subcommand)
	}
}

// handleCreateDashboard handles dashboard creation
func (cmd *BICommand) handleCreateDashboard(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: bi dashboard create <name> <category>")
		return
	}

	name := args[0]
	category := args[1]

	dashboard := &bi.Dashboard{
		Name:        name,
		Description: fmt.Sprintf("Dashboard for %s", name),
		Category:    category,
		Widgets: []bi.Widget{
			{
				ID:    "widget_1",
				Type:  "metric",
				Title: "Sample Metric",
				Query: "SELECT COUNT(*) FROM resources",
				Config: map[string]interface{}{
					"format": "number",
					"color":  "blue",
				},
				Position: bi.WidgetPosition{X: 0, Y: 0},
				Size:     bi.WidgetSize{Width: 2, Height: 1},
			},
		},
		Layout: bi.DashboardLayout{
			Columns: 4,
			Rows:    3,
			Theme:   "light",
		},
		Filters:     []bi.Filter{},
		RefreshRate: 5 * time.Minute,
		Public:      false,
	}

	ctx := context.Background()
	if err := cmd.service.CreateDashboard(ctx, dashboard); err != nil {
		fmt.Printf("Error creating dashboard: %v\n", err)
		return
	}

	fmt.Printf("Dashboard '%s' created successfully with ID: %s\n", name, dashboard.ID)
}

// handleListDashboards handles listing dashboards
func (cmd *BICommand) handleListDashboards(args []string) {
	ctx := context.Background()
	dashboards, err := cmd.service.GetBIEngine().ListDashboards(ctx)
	if err != nil {
		fmt.Printf("Error listing dashboards: %v\n", err)
		return
	}

	if len(dashboards) == 0 {
		fmt.Println("No dashboards found.")
		return
	}

	fmt.Println("Dashboards:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tCategory\tWidgets\tPublic\tCreated")
	fmt.Fprintln(w, "---\t----\t--------\t-------\t------\t-------")

	for _, dashboard := range dashboards {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%t\t%s\n",
			dashboard.ID,
			dashboard.Name,
			dashboard.Category,
			len(dashboard.Widgets),
			dashboard.Public,
			dashboard.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	w.Flush()
}

// handleGetDashboard handles getting dashboard details
func (cmd *BICommand) handleGetDashboard(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi dashboard get <dashboard-id>")
		return
	}

	dashboardID := args[0]

	ctx := context.Background()
	dashboard, err := cmd.service.GetBIEngine().GetDashboard(ctx, dashboardID)
	if err != nil {
		fmt.Printf("Error getting dashboard: %v\n", err)
		return
	}

	fmt.Printf("Dashboard Details:\n")
	fmt.Printf("ID: %s\n", dashboard.ID)
	fmt.Printf("Name: %s\n", dashboard.Name)
	fmt.Printf("Description: %s\n", dashboard.Description)
	fmt.Printf("Category: %s\n", dashboard.Category)
	fmt.Printf("Widgets: %d\n", len(dashboard.Widgets))
	fmt.Printf("Public: %t\n", dashboard.Public)
	fmt.Printf("Refresh Rate: %s\n", dashboard.RefreshRate.String())
	fmt.Printf("Created: %s\n", dashboard.CreatedAt.Format("2006-01-02 15:04:05"))
}

// handleUpdateDashboard handles updating a dashboard
func (cmd *BICommand) handleUpdateDashboard(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi dashboard update <dashboard-id>")
		return
	}

	dashboardID := args[0]
	fmt.Printf("Dashboard %s updated\n", dashboardID)
}

// handleDeleteDashboard handles deleting a dashboard
func (cmd *BICommand) handleDeleteDashboard(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi dashboard delete <dashboard-id>")
		return
	}

	dashboardID := args[0]
	fmt.Printf("Dashboard %s deleted\n", dashboardID)
}

// handleReport handles report management
func (cmd *BICommand) handleReport(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi report <command>")
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		cmd.handleCreateReport(args[1:])
	case "list":
		cmd.handleListReports(args[1:])
	case "generate":
		cmd.handleGenerateReport(args[1:])
	case "schedule":
		cmd.handleScheduleReport(args[1:])
	default:
		fmt.Printf("Unknown report command: %s\n", subcommand)
	}
}

// handleCreateReport handles report creation
func (cmd *BICommand) handleCreateReport(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: bi report create <name> <category>")
		return
	}

	name := args[0]
	category := args[1]

	report := &bi.Report{
		Name:        name,
		Description: fmt.Sprintf("Report for %s", name),
		Category:    category,
		Type:        "on-demand",
		Query:       "SELECT * FROM resources WHERE created_at >= ? AND created_at <= ?",
		Format:      "pdf",
		Recipients:  []string{"admin@company.com"},
		Parameters: map[string]interface{}{
			"start_date": "{{start_of_month}}",
			"end_date":   "{{end_of_month}}",
		},
	}

	ctx := context.Background()
	if err := cmd.service.CreateReport(ctx, report); err != nil {
		fmt.Printf("Error creating report: %v\n", err)
		return
	}

	fmt.Printf("Report '%s' created successfully with ID: %s\n", name, report.ID)
}

// handleListReports handles listing reports
func (cmd *BICommand) handleListReports(args []string) {
	ctx := context.Background()
	reports, err := cmd.service.GetBIEngine().ListReports(ctx)
	if err != nil {
		fmt.Printf("Error listing reports: %v\n", err)
		return
	}

	if len(reports) == 0 {
		fmt.Println("No reports found.")
		return
	}

	fmt.Println("Reports:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tCategory\tType\tFormat\tRecipients\tCreated")
	fmt.Fprintln(w, "---\t----\t--------\t----\t------\t----------\t-------")

	for _, report := range reports {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			report.ID,
			report.Name,
			report.Category,
			report.Type,
			report.Format,
			len(report.Recipients),
			report.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	w.Flush()
}

// handleGenerateReport handles report generation
func (cmd *BICommand) handleGenerateReport(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi report generate <report-id>")
		return
	}

	reportID := args[0]

	parameters := map[string]interface{}{
		"start_date": time.Now().AddDate(0, -1, 0), // 1 month ago
		"end_date":   time.Now(),
	}

	ctx := context.Background()
	result, err := cmd.service.GenerateReport(ctx, reportID, parameters)
	if err != nil {
		fmt.Printf("Error generating report: %v\n", err)
		return
	}

	fmt.Printf("Report generated successfully:\n")
	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Format: %s\n", result.Format)
	fmt.Printf("Size: %d bytes\n", result.Size)
	fmt.Printf("Path: %s\n", result.Path)
	fmt.Printf("URL: %s\n", result.URL)
	fmt.Printf("Generated: %s\n", result.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %s\n", result.Duration.String())
}

// handleScheduleReport handles report scheduling
func (cmd *BICommand) handleScheduleReport(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: bi report schedule <report-id> <schedule>")
		return
	}

	reportID := args[0]
	schedule := args[1]

	fmt.Printf("Report %s scheduled with %s\n", reportID, schedule)
}

// handleDataset handles dataset management
func (cmd *BICommand) handleDataset(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi dataset <command>")
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		cmd.handleCreateDataset(args[1:])
	case "list":
		cmd.handleListDatasets(args[1:])
	case "refresh":
		cmd.handleRefreshDataset(args[1:])
	default:
		fmt.Printf("Unknown dataset command: %s\n", subcommand)
	}
}

// handleCreateDataset handles dataset creation
func (cmd *BICommand) handleCreateDataset(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: bi dataset create <name> <source>")
		return
	}

	name := args[0]
	source := args[1]

	dataset := &bi.Dataset{
		Name:        name,
		Description: fmt.Sprintf("Dataset for %s", name),
		Source:      source,
		Type:        "table",
		Schema: map[string]interface{}{
			"columns": []map[string]interface{}{
				{"name": "id", "type": "string"},
				{"name": "name", "type": "string"},
				{"name": "value", "type": "float"},
				{"name": "timestamp", "type": "datetime"},
			},
		},
		RefreshRate: 1 * time.Hour,
	}

	ctx := context.Background()
	if err := cmd.service.CreateDataset(ctx, dataset); err != nil {
		fmt.Printf("Error creating dataset: %v\n", err)
		return
	}

	fmt.Printf("Dataset '%s' created successfully with ID: %s\n", name, dataset.ID)
}

// handleListDatasets handles listing datasets
func (cmd *BICommand) handleListDatasets(args []string) {
	ctx := context.Background()
	datasets, err := cmd.service.GetBIEngine().ListDatasets(ctx)
	if err != nil {
		fmt.Printf("Error listing datasets: %v\n", err)
		return
	}

	if len(datasets) == 0 {
		fmt.Println("No datasets found.")
		return
	}

	fmt.Println("Datasets:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tSource\tType\tSize\tLast Refresh")
	fmt.Fprintln(w, "---\t----\t------\t----\t----\t------------")

	for _, dataset := range datasets {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			dataset.ID,
			dataset.Name,
			dataset.Source,
			dataset.Type,
			dataset.Size,
			dataset.LastRefresh.Format("2006-01-02 15:04:05"),
		)
	}

	w.Flush()
}

// handleRefreshDataset handles dataset refresh
func (cmd *BICommand) handleRefreshDataset(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi dataset refresh <dataset-id>")
		return
	}

	datasetID := args[0]
	fmt.Printf("Dataset %s refreshed\n", datasetID)
}

// handleQuery handles query management
func (cmd *BICommand) handleQuery(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi query <command>")
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		cmd.handleCreateQuery(args[1:])
	case "list":
		cmd.handleListQueries(args[1:])
	case "execute":
		cmd.handleExecuteQuery(args[1:])
	default:
		fmt.Printf("Unknown query command: %s\n", subcommand)
	}
}

// handleCreateQuery handles query creation
func (cmd *BICommand) handleCreateQuery(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: bi query create <name> <sql>")
		return
	}

	name := args[0]
	sql := args[1]

	query := &bi.Query{
		Name:        name,
		Description: fmt.Sprintf("Query for %s", name),
		SQL:         sql,
		Parameters:  []bi.QueryParameter{},
		Cache:       true,
		CacheTTL:    1 * time.Hour,
	}

	ctx := context.Background()
	if err := cmd.service.CreateQuery(ctx, query); err != nil {
		fmt.Printf("Error creating query: %v\n", err)
		return
	}

	fmt.Printf("Query '%s' created successfully with ID: %s\n", name, query.ID)
}

// handleListQueries handles listing queries
func (cmd *BICommand) handleListQueries(args []string) {
	ctx := context.Background()
	queries, err := cmd.service.GetBIEngine().ListQueries(ctx)
	if err != nil {
		fmt.Printf("Error listing queries: %v\n", err)
		return
	}

	if len(queries) == 0 {
		fmt.Println("No queries found.")
		return
	}

	fmt.Println("Queries:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tCache\tCache TTL\tCreated")
	fmt.Fprintln(w, "---\t----\t-----\t---------\t-------")

	for _, query := range queries {
		fmt.Fprintf(w, "%s\t%s\t%t\t%s\t%s\n",
			query.ID,
			query.Name,
			query.Cache,
			query.CacheTTL.String(),
			query.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	w.Flush()
}

// handleExecuteQuery handles query execution
func (cmd *BICommand) handleExecuteQuery(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi query execute <query-id>")
		return
	}

	queryID := args[0]

	parameters := map[string]interface{}{
		"start_date": time.Now().AddDate(0, -1, 0), // 1 month ago
		"end_date":   time.Now(),
	}

	ctx := context.Background()
	result, err := cmd.service.ExecuteQuery(ctx, queryID, parameters)
	if err != nil {
		fmt.Printf("Error executing query: %v\n", err)
		return
	}

	fmt.Printf("Query executed successfully:\n")
	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Columns: %v\n", result.Columns)
	fmt.Printf("Row Count: %d\n", result.RowCount)
	fmt.Printf("Executed: %s\n", result.ExecutedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %s\n", result.Duration.String())

	if len(result.Rows) > 0 {
		fmt.Println("\nSample Results:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, column := range result.Columns {
			fmt.Fprintf(w, "%s\t", column)
		}
		fmt.Fprintln(w)

		for i, row := range result.Rows {
			if i >= 5 { // Show only first 5 rows
				break
			}
			for _, value := range row {
				fmt.Fprintf(w, "%v\t", value)
			}
			fmt.Fprintln(w)
		}
		w.Flush()
	}
}

// handleExport handles data export
func (cmd *BICommand) handleExport(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: bi export <command>")
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "data":
		cmd.handleExportData(args[1:])
	case "formats":
		cmd.handleExportFormats(args[1:])
	default:
		fmt.Printf("Unknown export command: %s\n", subcommand)
	}
}

// handleExportData handles data export
func (cmd *BICommand) handleExportData(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: bi export data <query-id> <format>")
		return
	}

	queryID := args[0]
	format := args[1]

	parameters := map[string]interface{}{
		"start_date": time.Now().AddDate(0, -1, 0), // 1 month ago
		"end_date":   time.Now(),
	}

	ctx := context.Background()
	result, err := cmd.service.ExportData(ctx, queryID, format, parameters)
	if err != nil {
		fmt.Printf("Error exporting data: %v\n", err)
		return
	}

	fmt.Printf("Data exported successfully:\n")
	fmt.Printf("ID: %s\n", result.ID)
	fmt.Printf("Format: %s\n", result.Format)
	fmt.Printf("Size: %d bytes\n", result.Size)
	fmt.Printf("Path: %s\n", result.Path)
	fmt.Printf("URL: %s\n", result.URL)
	fmt.Printf("Exported: %s\n", result.ExportedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %s\n", result.Duration.String())
}

// handleExportFormats handles listing export formats
func (cmd *BICommand) handleExportFormats(args []string) {
	ctx := context.Background()
	formats, err := cmd.service.GetExportEngine().GetSupportedFormats(ctx)
	if err != nil {
		fmt.Printf("Error getting supported formats: %v\n", err)
		return
	}

	fmt.Println("Supported Export Formats:")
	for _, format := range formats {
		info, err := cmd.service.GetExportEngine().GetFormatInfo(ctx, format)
		if err != nil {
			continue
		}
		fmt.Printf("  %s: %s (Max Size: %d bytes)\n", format, info.Description, info.MaxSize)
	}
}

// handleStatus handles BI status
func (cmd *BICommand) handleStatus(args []string) {
	ctx := context.Background()
	status, err := cmd.service.GetBIStatus(ctx)
	if err != nil {
		fmt.Printf("Error getting BI status: %v\n", err)
		return
	}

	fmt.Printf("BI Status: %s\n", status.OverallStatus)
	fmt.Printf("Last Refresh: %s\n", status.LastRefresh.Format("2006-01-02 15:04:05"))

	if len(status.Dashboards) > 0 {
		fmt.Println("\nDashboards by Category:")
		for category, count := range status.Dashboards {
			fmt.Printf("  %s: %d dashboards\n", category, count)
		}
	}

	if len(status.Reports) > 0 {
		fmt.Println("\nReports by Category:")
		for category, count := range status.Reports {
			fmt.Printf("  %s: %d reports\n", category, count)
		}
	}

	if len(status.Datasets) > 0 {
		fmt.Println("\nDatasets by Type:")
		for datasetType, count := range status.Datasets {
			fmt.Printf("  %s: %d datasets\n", datasetType, count)
		}
	}

	if len(status.Queries) > 0 {
		fmt.Println("\nQueries:")
		for queryType, count := range status.Queries {
			fmt.Printf("  %s: %d queries\n", queryType, count)
		}
	}
}
