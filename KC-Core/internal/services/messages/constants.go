// Package messages provides message listing service implementation for gRPC layer.
package messages

// Export format constants - canonical location (domain layer).
// pkg/grpc/export_constants.go re-exports these for backward compatibility.
const (
	// ExportFormatJSON indicates JSON export format
	ExportFormatJSON = "json"
	// ExportFormatCSV indicates CSV export format
	ExportFormatCSV = "csv"
	// NOTE: xlsx is NOT supported - do not add without implementation

	// ExportMaxLimit is the maximum number of records for export operations
	ExportMaxLimit = 10000
)

// ExportCSVHeaders defines the column order for CSV exports.
var ExportCSVHeaders = []string{
	"id",
	"ep_eui",
	"bs_eui",
	"rx_time",
	"packet_cnt",
	"snr",
	"rssi",
	"user_data",
}

// IsValidExportFormat checks if the given format is supported.
func IsValidExportFormat(format string) bool {
	switch format {
	case ExportFormatJSON, ExportFormatCSV:
		return true
	default:
		return false
	}
}

// SupportedExportFormats returns all supported export formats.
func SupportedExportFormats() []string {
	return []string{ExportFormatJSON, ExportFormatCSV}
}
