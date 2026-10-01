// Package main provides a MIOTY log file ingestion utility for importing historical uplink data.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/pkg/keycrypto"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Local development database defaults for this ingestion utility.
const (
	// defaultDBPort matches the Docker Compose host mapping for PostgreSQL.
	defaultDBPort = 5433

	// summaryCountProbeLimit fetches a single row; only the returned total
	// count is used for the ingestion summary.
	summaryCountProbeLimit = 1
)

// euiHexLength is the length of an EUI64 in hex characters.
const euiHexLength = 16

// Development database connection defaults (mirrors docker-compose.dev.yml).
const (
	defaultDBHost     = "localhost"
	defaultDBName     = "kilocenter"
	defaultDBUser     = "kilocenter"
	defaultDBPassword = "changeme"
	defaultDBSSLMode  = "disable"
)

// Base station log line markers parsed from the MIOTY log format.
const (
	markerDataReceived = "data received from end point"
	markerEndPoint     = "end point "
	markerCnt          = "cnt "
	markerRSSI         = "RSSI "
	markerSNR          = "SNR "
	markerEqSNR        = "eqSNR "
)

// progressInterval throttles progress output to every N processed messages.
const progressInterval = 100

func main() {
	// Define command-line flags
	var (
		baseStationEUI = flag.String("bs-eui", "0000000000000001", "Base station EUI (16 hex characters)")
		tenantIDFlag   = flag.Int64("tenant-id", 1, "Tenant ID")
	)
	flag.Parse()

	// Check for log file argument
	if flag.NArg() < 1 {
		fmt.Println("Usage: mioty-ingest [options] <logfile>")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nIngests MIOTY log files into the KiloCenter database")
		os.Exit(1)
	}

	logFile := flag.Arg(0)
	tenantID := *tenantIDFlag

	// Validate base station EUI format
	if len(*baseStationEUI) != euiHexLength {
		log.Fatalf("Invalid base station EUI: must be 16 hex characters")
	}

	// Connect to database using postgres storage
	storageOpts := postgres.Options{
		Host:     defaultDBHost,
		Port:     defaultDBPort,
		Database: defaultDBName,
		Username: defaultDBUser,
		Password: defaultDBPassword,
		SSLMode:  defaultDBSSLMode,
	}

	cipher, err := keycrypto.NewCipherFromMasterKey(os.Getenv("KILOCENTER_MASTER_KEY"))
	if err != nil {
		log.Fatalf("Failed to build key-material cipher from KILOCENTER_MASTER_KEY: %v", err)
	}

	store, err := postgres.New(storageOpts, cipher)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("[WARN] Failed to close database connection: %v", err)
		}
	}()

	fmt.Println("PASS: Database connection successful")
	repos := postgres.NewRepositories(store)

	// Open log file - sanitize path to prevent path traversal
	logFile = filepath.Clean(logFile)
	absLogFile, err := filepath.Abs(logFile)
	if err != nil {
		log.Fatalf("Failed to resolve log file path: %v", err)
	}
	if strings.Contains(absLogFile, "..") {
		log.Fatalf("Log file path contains path traversal")
	}
	file, err := os.Open(logFile)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("[WARN] Failed to close log file: %v", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	ctx := context.Background() // context-root: process
	var processedCount int
	var errorCount int

	fmt.Printf("Processing log file: %s\n", logFile)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip non-data lines
		if !strings.Contains(line, markerDataReceived) {
			continue
		}

		// Simple parsing for MIOTY log format
		// Format: "DD HH:MM:SS INF data received from end point XX-XX-XX-XX-XX-XX-XX-XX, cnt N, RSSI X dBm, SNR Y dB, eqSNR Z dB"

		// Extract endpoint EUI
		epStart := strings.Index(line, markerEndPoint) + len(markerEndPoint)
		epEnd := strings.Index(line[epStart:], ",")
		if epEnd == -1 {
			errorCount++
			continue
		}
		epEUI := line[epStart : epStart+epEnd]
		epEUI = strings.ReplaceAll(epEUI, "-", "")

		// Extract packet count
		cntStart := strings.Index(line, markerCnt) + len(markerCnt)
		cntEnd := strings.Index(line[cntStart:], ",")
		var packetCnt int
		if n, err := fmt.Sscanf(line[cntStart:cntStart+cntEnd], "%d", &packetCnt); n != 1 || err != nil {
			log.Printf("[FAIL] Failed to parse packet count: %v", err)
			errorCount++
			continue
		}

		// Extract RSSI
		rssiStart := strings.Index(line, markerRSSI) + len(markerRSSI)
		rssiEnd := strings.Index(line[rssiStart:], " ")
		var rssi float64
		if n, err := fmt.Sscanf(line[rssiStart:rssiStart+rssiEnd], "%f", &rssi); n != 1 || err != nil {
			log.Printf("[FAIL] Failed to parse RSSI: %v", err)
			errorCount++
			continue
		}

		// Extract SNR
		snrStart := strings.Index(line, markerSNR) + len(markerSNR)
		snrEnd := strings.Index(line[snrStart:], " ")
		var snr float64
		if n, err := fmt.Sscanf(line[snrStart:snrStart+snrEnd], "%f", &snr); n != 1 || err != nil {
			log.Printf("[FAIL] Failed to parse SNR: %v", err)
			errorCount++
			continue
		}

		eqSnr := optionalEqSNR(line)

		// Convert EUI strings to uint64
		epEuiInt, err := strconv.ParseUint(epEUI, 16, 64)
		if err != nil {
			log.Printf("[FAIL] Invalid endpoint EUI: %s", epEUI)
			errorCount++
			continue
		}

		bsEuiInt, err := strconv.ParseUint(*baseStationEUI, 16, 64)
		if err != nil {
			log.Printf("[FAIL] Invalid base station EUI: %s", *baseStationEUI)
			errorCount++
			continue
		}

		// Validate OpID fits in uint64 (int is always smaller than uint64 max on 64-bit systems)
		if processedCount < 0 {
			log.Printf("[FAIL] OpID counter overflow, skipping")
			errorCount++
			continue
		}

		// Validate PacketCnt won't exceed uint32 max
		if packetCnt > math.MaxUint32 {
			log.Printf("[FAIL] PacketCnt exceeds uint32 max, skipping")
			errorCount++
			continue
		}

		// The log line has no date, so the import moment stands in for the reception time.
		importedAt := time.Now()
		msg := &mioty.ULDataMessage{
			ID:          uuid.New().String(),
			CommandType: mioty.CmdULData,
			OpId:        int64(processedCount + 1), // Positive for ingest (simulates BS)
			EpEui:       epEuiInt,
			BsEui:       bsEuiInt,
			RxTime:      importedAt.UnixNano(),
			PacketCnt:   uint32(packetCnt), //nolint:gosec // G115: validated above on line 162

			SNR:         snr,
			RSSI:        rssi,
			UserData:    []byte{},
			DlOpen:      false,
			ResponseExp: false,
			DlAck:       false,
			TenantID:    tenantID,
			ReceivedAt:  importedAt,
		}

		msg.EqSnr = eqSnr

		msg.BaseStations = []mioty.BaseStationReception{{
			BsEui: msg.BsEui, RxTime: msg.RxTime, Snr: msg.SNR, Rssi: msg.RSSI, EqSnr: msg.EqSnr,
		}}
		// Historical imports are classified like live traffic but never fan out.
		_, err = repos.UplinkStore.Persist(ctx, models.UplinkPersistRequest{
			Message: msg,
			Window:  time.Duration(config.DefaultProtocolDuplicateWindow) * time.Second,
		})
		if err != nil {
			log.Printf("[FAIL] Failed to store message: %v", err)
			errorCount++
			continue
		}

		processedCount++
		if processedCount%progressInterval == 0 {
			fmt.Printf("  Processed %d messages...\n", processedCount)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading log file: %v", err)
	}

	fmt.Printf("\nPASS: Ingestion complete!\n")
	fmt.Printf("   Processed: %d messages\n", processedCount)
	fmt.Printf("   Errors: %d\n", errorCount)

	// Show summary - query new messages table
	var count int64
	filter := mioty.ULDataMessageFilter{
		TenantID: tenantID,
		Limit:    summaryCountProbeLimit,
		Offset:   0,
	}
	_, count, err = repos.Messages.ListULDataMessages(ctx, filter)
	if err != nil {
		log.Printf("Failed to get message count: %v", err)
	} else {
		fmt.Printf("   Total messages in database: %d\n", count)
	}
}

// optionalEqSNR returns the line's eqSNR; the field is optional, so a line
// without one, or with a malformed one, yields nil.
func optionalEqSNR(line string) *float64 {
	start := strings.Index(line, markerEqSNR)
	if start < 0 {
		return nil
	}
	field := line[start+len(markerEqSNR):]
	if end := strings.Index(field, " "); end >= 0 {
		field = field[:end]
	}
	value, err := strconv.ParseFloat(field, 64)
	if err != nil {
		log.Printf("[WARN] Ignoring malformed eqSNR %q: %v", field, err)
		return nil
	}
	return &value
}
