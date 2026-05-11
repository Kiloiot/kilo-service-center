package bssci

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// MessageDeduplicator handles deduplication of MIOTY messages
type MessageDeduplicator struct {
	mu              sync.RWMutex
	recentMessages  map[string]*MessageRecord
	windowDuration  time.Duration
	cleanupInterval time.Duration
	stopCh          chan struct{}
}

// MessageRecord stores information about a received message
type MessageRecord struct {
	EndpointEUI      uint64
	PacketCounter    uint32
	MessageHash      string
	FirstReceivedAt  time.Time
	FirstBaseStation uint64
	BaseStations     map[uint64]ReceptionInfo // Map of base station EUI to reception info
	DuplicateCount   int
}

// ReceptionInfo stores reception details from a base station per SCACI §3.8.1
type ReceptionInfo struct {
	ReceivedAt    time.Time
	RSSI          float64
	SNR           float64
	EqSNR         *float64 // Optional - nil if not present (0.0 is valid)
	BaseStationID uint64
	// SCACI §3.8.1 fields from BSSCI UL payload:
	RxTime     int64             // Unix UTC ns (mandatory)
	RxDuration *int64            // Optional - nil if not present
	Profile    *string           // Optional
	Mode       *string           // Optional
	Subpackets *mioty.Subpackets // Optional
}

// NewMessageDeduplicator creates a new deduplicator
func NewMessageDeduplicator(windowDuration time.Duration) *MessageDeduplicator {
	if windowDuration <= 0 {
		windowDuration = 5 * time.Minute // Default 5 minute window per MIOTY spec
	}

	md := &MessageDeduplicator{
		recentMessages:  make(map[string]*MessageRecord),
		windowDuration:  windowDuration,
		cleanupInterval: windowDuration / 2,
		stopCh:          make(chan struct{}),
	}

	// Start cleanup routine
	go md.cleanupRoutine()

	return md
}

// Stop stops the deduplicator cleanup routine
func (md *MessageDeduplicator) Stop() {
	close(md.stopCh)
}

// WindowDuration returns the deduplication window duration for time-window enforcement
func (md *MessageDeduplicator) WindowDuration() time.Duration {
	return md.windowDuration
}

// CheckAndRecord checks if a message is a duplicate and records it
// Returns: isDuplicate, record, error
//
// Parameters:
//   - endpointEUI: The endpoint's EUI (mandatory)
//   - packetCounter: The packet counter (mandatory)
//   - baseStationEUI: The base station's EUI (mandatory)
//   - userData: The user data payload (mandatory)
//   - rssi: RSSI value (mandatory)
//   - snr: SNR value (mandatory)
//   - eqSnr: Optional equivalent SNR - pass nil if not present (0.0 is valid)
//   - rxTime: Reception time in Unix UTC ns (mandatory)
//   - rxDuration: Optional reception duration - pass nil if not present
//   - profile: Optional profile string - pass nil if not present
//   - mode: Optional mode string - pass nil if not present
//   - subpackets: Optional subpackets - pass nil if not present
func (md *MessageDeduplicator) CheckAndRecord(
	endpointEUI uint64,
	packetCounter uint32,
	baseStationEUI uint64,
	userData []byte,
	rssi, snr float64,
	eqSnr *float64,
	rxTime int64,
	rxDuration *int64,
	profile *string,
	mode *string,
	subpackets *mioty.Subpackets,
) (bool, *MessageRecord, error) {

	// Create deduplication key: endpoint EUI + packet counter
	dedupKey := fmt.Sprintf("%016X:%08X", endpointEUI, packetCounter)

	// Create message hash for additional validation
	hasher := sha256.New()
	_, _ = fmt.Fprintf(hasher, "%016X", endpointEUI)
	_, _ = fmt.Fprintf(hasher, "%08X", packetCounter)
	hasher.Write(userData)
	messageHash := hex.EncodeToString(hasher.Sum(nil))

	md.mu.Lock()
	defer md.mu.Unlock()

	now := time.Now()

	// Check if we've seen this message before
	if record, exists := md.recentMessages[dedupKey]; exists {
		// Verify message hash matches (additional safety check)
		if record.MessageHash != messageHash {
			return false, nil, fmt.Errorf("%s", ResolveErrorMessage(errPacketCounterCollision))
		}

		// Check if within deduplication window
		if now.Sub(record.FirstReceivedAt) > md.windowDuration {
			// Outside window, treat as new message (possible counter rollover)
			delete(md.recentMessages, dedupKey)
		} else {
			// It's a duplicate within the window
			record.DuplicateCount++

			// Add this base station's reception info with SCACI §3.8.1 fields
			record.BaseStations[baseStationEUI] = ReceptionInfo{
				ReceivedAt:    now,
				RSSI:          rssi,
				SNR:           snr,
				EqSNR:         eqSnr,
				BaseStationID: baseStationEUI,
				RxTime:        rxTime,
				RxDuration:    rxDuration,
				Profile:       profile,
				Mode:          mode,
				Subpackets:    subpackets,
			}

			return true, record, nil
		}
	}

	// New message - create record
	record := &MessageRecord{
		EndpointEUI:      endpointEUI,
		PacketCounter:    packetCounter,
		MessageHash:      messageHash,
		FirstReceivedAt:  now,
		FirstBaseStation: baseStationEUI,
		BaseStations:     make(map[uint64]ReceptionInfo),
		DuplicateCount:   0,
	}

	// Add first base station's reception info with SCACI §3.8.1 fields
	record.BaseStations[baseStationEUI] = ReceptionInfo{
		ReceivedAt:    now,
		RSSI:          rssi,
		SNR:           snr,
		EqSNR:         eqSnr,
		BaseStationID: baseStationEUI,
		RxTime:        rxTime,
		RxDuration:    rxDuration,
		Profile:       profile,
		Mode:          mode,
		Subpackets:    subpackets,
	}

	// Store the record
	md.recentMessages[dedupKey] = record

	return false, record, nil
}

// GetBestReception returns the base station with the best signal quality
func (record *MessageRecord) GetBestReception() (uint64, *ReceptionInfo) {
	var bestBS uint64
	var bestInfo *ReceptionInfo
	var bestSNR float64 = -999

	for bsEUI, info := range record.BaseStations {
		if info.SNR > bestSNR {
			bestSNR = info.SNR
			bestBS = bsEUI
			infoCopy := info
			bestInfo = &infoCopy
		}
	}

	return bestBS, bestInfo
}

// ConvertToBaseStationReceptions converts dedup record to SCACI-compatible array per §3.8.1
func (record *MessageRecord) ConvertToBaseStationReceptions() []mioty.BaseStationReception {
	receptions := make([]mioty.BaseStationReception, 0, len(record.BaseStations))

	for bsEUI, info := range record.BaseStations {
		reception := mioty.BaseStationReception{
			BsEui:      bsEUI,
			RxTime:     info.RxTime,
			Snr:        info.SNR,
			Rssi:       info.RSSI,
			EqSnr:      info.EqSNR,
			RxDuration: info.RxDuration,
			Profile:    info.Profile,
			Mode:       info.Mode,
			Subpackets: info.Subpackets,
		}
		receptions = append(receptions, reception)
	}

	return receptions
}

// cleanupRoutine periodically removes old records outside the deduplication window
func (md *MessageDeduplicator) cleanupRoutine() {
	ticker := time.NewTicker(md.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			md.cleanup()
		case <-md.stopCh:
			return
		}
	}
}

// cleanup removes old records
func (md *MessageDeduplicator) cleanup() {
	md.mu.Lock()
	defer md.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-md.windowDuration)

	for key, record := range md.recentMessages {
		if record.FirstReceivedAt.Before(cutoff) {
			delete(md.recentMessages, key)
		}
	}
}

// GetStatistics returns deduplication statistics
func (md *MessageDeduplicator) GetStatistics() map[string]interface{} {
	md.mu.RLock()
	defer md.mu.RUnlock()

	totalMessages := len(md.recentMessages)
	totalDuplicates := 0
	multiBaseStationCount := 0

	for _, record := range md.recentMessages {
		totalDuplicates += record.DuplicateCount
		if len(record.BaseStations) > 1 {
			multiBaseStationCount++
		}
	}

	return map[string]interface{}{
		"total_unique_messages":      totalMessages,
		"total_duplicates":           totalDuplicates,
		"multi_basestation_messages": multiBaseStationCount,
		"window_duration_seconds":    md.windowDuration.Seconds(),
	}
}
