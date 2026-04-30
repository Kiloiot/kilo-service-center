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

	"github.com/google/uuid"
	"github.com/kilocenter/KC-Core/pkg/config"
	"github.com/kilocenter/KC-DB/storage/mioty"
	"github.com/kilocenter/KC-DB/storage/postgres"
	_ "github.com/lib/pq"
)

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
	if len(*baseStationEUI) != 16 {
		log.Fatalf("Invalid base station EUI: must be 16 hex characters")
	}

	// Connect to database using postgres storage
	cfg := config.StorageConfig{
		Host:     "localhost",
		Port:     5433,
		Database: "kilocenter",
		Username: "kilocenter",
		Password: "changeme",
		SSLMode:  "disable",
	}

	store, err := postgres.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() { _ = store.Close() }()

	fmt.Println("PASS: Database connection successful")

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
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	ctx := context.Background()
	processedCount := 0
	errorCount := 0

	fmt.Printf("Processing log file: %s\n", logFile)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip non-data lines
		if !strings.Contains(line, "data received from end point") {
			continue
		}

		// Simple parsing for MIOTY log format
		// Format: "DD HH:MM:SS INF data received from end point XX-XX-XX-XX-XX-XX-XX-XX, cnt N, RSSI X dBm, SNR Y dB, eqSNR Z dB"

		// Extract endpoint EUI
		epStart := strings.Index(line, "end point ") + 10
		epEnd := strings.Index(line[epStart:], ",")
		if epEnd == -1 {
			errorCount++
			continue
		}
		epEUI := line[epStart : epStart+epEnd]
		epEUI = strings.ReplaceAll(epEUI, "-", "")

		// Extract packet count
		cntStart := strings.Index(line, "cnt ") + 4
		cntEnd := strings.Index(line[cntStart:], ",")
		var packetCnt int
		if n, err := fmt.Sscanf(line[cntStart:cntStart+cntEnd], "%d", &packetCnt); n != 1 || err != nil {
			log.Printf("[FAIL] Failed to parse packet count: %v", err)
			errorCount++
			continue
		}

		// Extract RSSI
		rssiStart := strings.Index(line, "RSSI ") + 5
		rssiEnd := strings.Index(line[rssiStart:], " ")
		var rssi float64
		if n, err := fmt.Sscanf(line[rssiStart:rssiStart+rssiEnd], "%f", &rssi); n != 1 || err != nil {
			log.Printf("[FAIL] Failed to parse RSSI: %v", err)
			errorCount++
			continue
		}

		// Extract SNR
		snrStart := strings.Index(line, "SNR ") + 4
		snrEnd := strings.Index(line[snrStart:], " ")
		var snr float64
		if n, err := fmt.Sscanf(line[snrStart:snrStart+snrEnd], "%f", &snr); n != 1 || err != nil {
			log.Printf("[FAIL] Failed to parse SNR: %v", err)
			errorCount++
			continue
		}

		// Extract eqSNR (optional field, ignore parse errors)
		eqSnrStart := strings.Index(line, "eqSNR ") + 6
		eqSnrEnd := strings.Index(line[eqSnrStart:], " ")
		var eqSnr float64
		_, _ = fmt.Sscanf(line[eqSnrStart:eqSnrStart+eqSnrEnd], "%f", &eqSnr)

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

		// Create MIOTY UL Data message
		msg := &mioty.ULDataMessage{
			ID:          uuid.New().String(),
			CommandType: "ulData",
			OpId:        int64(processedCount + 1), // Positive for ingest (simulates BS)
			EpEui:       epEuiInt,
			BsEui:       bsEuiInt,
			RxTime:      time.Now().UnixNano(),
			PacketCnt:   uint32(packetCnt), //nolint:gosec // G115: validated above on line 162

			SNR:         snr,
			RSSI:        rssi,
			UserData:    []byte{},
			DlOpen:      false,
			ResponseExp: false,
			DlAck:       false,
			TenantID:    tenantID,
			ReceivedAt:  time.Now(),
		}

		// Add optional eqSnr if present
		if eqSnr != 0 {
			msg.EqSnr = &eqSnr
		}

		// Store message using message repository
		err = store.CreateULDataMessage(ctx, msg)
		if err != nil {
			log.Printf("[FAIL] Failed to store message: %v", err)
			errorCount++
			continue
		}

		processedCount++
		if processedCount%100 == 0 {
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
		Limit:    1, // Just to get count
		Offset:   0,
	}
	_, count, err = store.ListULDataMessages(ctx, filter)
	if err != nil {
		log.Printf("Failed to get message count: %v", err)
	} else {
		fmt.Printf("   Total messages in database: %d\n", count)
	}
}
