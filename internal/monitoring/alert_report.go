package monitoring

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/guarzo/pkmgradegap/internal/report"
)

// AlertReport contains alert data and metadata for export
type AlertReport struct {
	Metadata AlertReportMetadata `json:"metadata"`
	Alerts   []Alert             `json:"alerts"`
}

// AlertReportMetadata contains report-level information
type AlertReportMetadata struct {
	GeneratedAt     time.Time   `json:"generated_at"`
	OldSnapshotPath string      `json:"old_snapshot_path"`
	NewSnapshotPath string      `json:"new_snapshot_path"`
	OldSnapshotTime time.Time   `json:"old_snapshot_time"`
	NewSnapshotTime time.Time   `json:"new_snapshot_time"`
	TotalAlerts     int         `json:"total_alerts"`
	HighSeverity    int         `json:"high_severity"`
	MediumSeverity  int         `json:"medium_severity"`
	LowSeverity     int         `json:"low_severity"`
	AlertConfig     AlertConfig `json:"alert_config"`
}

// GenerateAlertReport creates a comprehensive alert report with metadata
func GenerateAlertReport(alerts []Alert, oldSnapshot, newSnapshot *Snapshot, oldPath, newPath string, config AlertConfig) *AlertReport {
	metadata := AlertReportMetadata{
		GeneratedAt:     time.Now(),
		OldSnapshotPath: oldPath,
		NewSnapshotPath: newPath,
		OldSnapshotTime: oldSnapshot.Timestamp,
		NewSnapshotTime: newSnapshot.Timestamp,
		TotalAlerts:     len(alerts),
		AlertConfig:     config,
	}

	// Count alerts by severity
	for _, alert := range alerts {
		switch alert.Severity {
		case "HIGH":
			metadata.HighSeverity++
		case "MEDIUM":
			metadata.MediumSeverity++
		case "LOW":
			metadata.LowSeverity++
		}
	}

	return &AlertReport{
		Metadata: metadata,
		Alerts:   alerts,
	}
}

// ExportToCSV exports the alert report to a CSV file
func (ar *AlertReport) ExportToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write metadata header
	metadataHeaders := []string{
		"Report Generated", ar.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"),
		"Old Snapshot", ar.Metadata.OldSnapshotPath,
		"New Snapshot", ar.Metadata.NewSnapshotPath,
		"Time Range", fmt.Sprintf("%s to %s",
			ar.Metadata.OldSnapshotTime.Format("2006-01-02 15:04:05"),
			ar.Metadata.NewSnapshotTime.Format("2006-01-02 15:04:05")),
		"Total Alerts", strconv.Itoa(ar.Metadata.TotalAlerts),
		"High Severity", strconv.Itoa(ar.Metadata.HighSeverity),
		"Medium Severity", strconv.Itoa(ar.Metadata.MediumSeverity),
		"Low Severity", strconv.Itoa(ar.Metadata.LowSeverity),
	}

	// Write metadata as key-value pairs
	for i := 0; i < len(metadataHeaders); i += 2 {
		row := report.EscapeCSVRow([]string{metadataHeaders[i], metadataHeaders[i+1]})
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("writing metadata: %w", err)
		}
	}

	// Write empty line separator
	if err := writer.Write([]string{}); err != nil {
		return fmt.Errorf("writing separator: %w", err)
	}

	// Write alert data headers
	headers := []string{
		"Alert Type",
		"Severity",
		"Timestamp",
		"Card Name",
		"Set Name",
		"Card Number",
		"Message",
		"Old Price",
		"New Price",
		"Price Change USD",
		"Price Change Percent",
		"Action Items",
		"Additional Details",
	}

	safeHeaders := report.EscapeCSVRow(headers)
	if err := writer.Write(safeHeaders); err != nil {
		return fmt.Errorf("writing headers: %w", err)
	}

	// Write alert data
	for _, alert := range ar.Alerts {
		row := []string{
			string(alert.Type),
			alert.Severity,
			alert.Timestamp.Format("2006-01-02 15:04:05"),
			alert.Card.Name,
			alert.Card.SetName,
			alert.Card.Number,
			alert.Message,
		}

		// Extract price information from details if available
		if details := alert.Details; details != nil {
			if oldPrice, ok := details["old_price"].(float64); ok {
				row = append(row, fmt.Sprintf("%.2f", oldPrice))
			} else {
				row = append(row, "")
			}

			if newPrice, ok := details["new_price"].(float64); ok {
				row = append(row, fmt.Sprintf("%.2f", newPrice))
			} else {
				row = append(row, "")
			}

			if deltaUSD, ok := details["delta_usd"].(float64); ok {
				row = append(row, fmt.Sprintf("%.2f", deltaUSD))
			} else {
				row = append(row, "")
			}

			if deltaPct, ok := details["delta_pct"].(float64); ok {
				row = append(row, fmt.Sprintf("%.2f", deltaPct))
			} else {
				row = append(row, "")
			}
		} else {
			// Empty price fields
			row = append(row, "", "", "", "")
		}

		// Join action items
		actionItems := ""
		if len(alert.ActionItems) > 0 {
			actionItems = fmt.Sprintf("%v", alert.ActionItems)
		}
		row = append(row, actionItems)

		// Additional details as JSON-like string
		additionalDetails := ""
		if alert.Details != nil {
			detailStrings := []string{}
			for key, value := range alert.Details {
				if key != "old_price" && key != "new_price" && key != "delta_usd" && key != "delta_pct" {
					detailStrings = append(detailStrings, fmt.Sprintf("%s: %v", key, value))
				}
			}
			if len(detailStrings) > 0 {
				additionalDetails = fmt.Sprintf("{%v}", detailStrings)
			}
		}
		row = append(row, additionalDetails)

		safeRow := report.EscapeCSVRow(row)
		if err := writer.Write(safeRow); err != nil {
			return fmt.Errorf("writing alert row: %w", err)
		}
	}

	return nil
}

// FormatAlertReport creates a human-readable string representation of the alert report
func FormatAlertReport(report *AlertReport) string {
	output := fmt.Sprintf("PRICE ALERTS REPORT\n")
	output += fmt.Sprintf("==================\n\n")

	// Report metadata
	output += fmt.Sprintf("Generated: %s\n", report.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("Time Range: %s to %s\n",
		report.Metadata.OldSnapshotTime.Format("2006-01-02 15:04:05"),
		report.Metadata.NewSnapshotTime.Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("Total Alerts: %d\n", report.Metadata.TotalAlerts)
	output += fmt.Sprintf("Severity Breakdown: %d High, %d Medium, %d Low\n\n",
		report.Metadata.HighSeverity, report.Metadata.MediumSeverity, report.Metadata.LowSeverity)

	// Configuration used
	output += fmt.Sprintf("Alert Configuration:\n")
	output += fmt.Sprintf("- Price Drop Threshold: %.1f%% or $%.2f\n",
		report.Metadata.AlertConfig.PriceDropThresholdPct, report.Metadata.AlertConfig.PriceDropThresholdUSD)
	output += fmt.Sprintf("- Opportunity ROI Threshold: %.1f%%\n", report.Metadata.AlertConfig.OpportunityThresholdROI)
	if report.Metadata.AlertConfig.VolatilityHighThreshold > 0 {
		output += fmt.Sprintf("- High Volatility Threshold: %.1f%%\n", report.Metadata.AlertConfig.VolatilityHighThreshold)
	}
	if report.Metadata.AlertConfig.VolatilityLowThreshold > 0 {
		output += fmt.Sprintf("- Low Volatility Threshold: %.1f%%\n", report.Metadata.AlertConfig.VolatilityLowThreshold)
	}
	if report.Metadata.AlertConfig.MinSeverity != "" {
		output += fmt.Sprintf("- Minimum Severity: %s\n", report.Metadata.AlertConfig.MinSeverity)
	}
	output += "\n"

	// Individual alerts
	if len(report.Alerts) == 0 {
		output += "No alerts found.\n"
	} else {
		output += "ALERTS:\n"
		output += "=======\n"
		for _, alert := range report.Alerts {
			output += FormatAlert(alert)
		}
	}

	return output
}
