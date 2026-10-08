// Package diagnostics assembles the server-admin diagnostics bundle: a zip of
// allowlisted, projected views of the running service center.
package diagnostics

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/version"
)

// Archive entry names; the bundle contains exactly these files.
const (
	EntryRelease       = "release.json"
	EntryConfig        = "config.json"
	EntryEvents        = "events.json"
	EntrySCACISessions = "scaci_sessions.json"
	EntryBSSCISessions = "bssci_sessions.json"
	EntrySchema        = "schema.json"
)

// Entries lists the archive members in the order they are written.
var Entries = []string{EntryRelease, EntryConfig, EntryEvents, EntrySCACISessions, EntryBSSCISessions, EntrySchema}

// Errors the handler maps onto gRPC statuses.
var (
	ErrBundleTooLarge = errors.New("diagnostics bundle exceeds the size limit")
)

const (
	errFmtTooLarge = "%w: %d bytes"
	errCtxRelease  = "read release info"
	errCtxEvents   = "list events"
	errCtxSessions = "read SCACI session statistics"
	errCtxArchive  = "write archive"
)

// EventLister returns the newest events of a tenant with projected details.
type EventLister interface {
	List(ctx context.Context, tenantID int64, filters *grpcservices.EventFilters, limit, offset int) ([]*grpcservices.Event, int64, error)
}

// SCACISessionSummarizer reads the per-tenant session statistics.
type SCACISessionSummarizer interface {
	GetSessionStatistics(ctx context.Context, tenantID int64) (*models.SCACISessionStatistics, error)
}

// BSSCISessionDirectory snapshots the live base station sessions.
type BSSCISessionDirectory interface {
	GetConnectedSessions() []map[string]interface{}
}

// ReleaseReader loads the embedded release manifest.
type ReleaseReader func() (*version.Info, error)

// Limits bound the bundle; they come from named configuration constants.
type Limits struct {
	MaxBundleBytes int
	MaxEvents      int
	MaxSessions    int
	Timeout        time.Duration
}

// Deps wires the bundle builder.
type Deps struct {
	Release    ReleaseReader
	Config     *config.Config
	Events     EventLister
	SCACI      SCACISessionSummarizer
	BSSCI      BSSCISessionDirectory
	Limits     Limits
	Clock      clock.Clock
	Log        logger.Logger
	ServerName string
}

// Bundle is the assembled archive.
type Bundle struct {
	Archive     []byte
	Filename    string
	ContentType string
	GeneratedAt time.Time
}

// Service builds diagnostics bundles.
type Service struct {
	release    ReleaseReader
	cfg        *config.Config
	events     EventLister
	scaci      SCACISessionSummarizer
	bssci      BSSCISessionDirectory
	limits     Limits
	clock      clock.Clock
	logger     logger.Logger
	serverName string
}

// New creates the bundle builder.
func New(d Deps) *Service {
	return &Service{release: d.Release, cfg: d.Config, events: d.Events, scaci: d.SCACI, bssci: d.BSSCI, limits: d.Limits, clock: d.Clock, logger: d.Log, serverName: d.ServerName}
}

// bsSessionSummary is what the bundle says about one base station session.
type bsSessionSummary struct {
	BsEui             string `json:"bsEui"`
	Connected         bool   `json:"connected"`
	HandshakeComplete bool   `json:"handshakeComplete"`
	NegotiatedVersion string `json:"negotiatedVersion"`
	Vendor            string `json:"vendor"`
	Model             string `json:"model"`
	Name              string `json:"name"`
	ClientVersion     string `json:"clientVersion"`
}

type bsSessionsEntry struct {
	TotalConnected int                `json:"totalConnected"`
	Sessions       []bsSessionSummary `json:"sessions"`
}

type schemaEntry struct {
	SchemaVersion int    `json:"schemaVersion"`
	Version       string `json:"version"`
}

type releaseEntry struct {
	*version.Info
	ServerName  string    `json:"serverName"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// Build assembles the bundle for the calling tenant; every entry is a
// projection, and the caller's tenant bounds the events and sessions.
func (s *Service) Build(ctx context.Context, tenantID int64) (*Bundle, error) {
	ctx, cancel := context.WithTimeout(ctx, s.limits.Timeout)
	defer cancel()
	now := s.clock.Now()

	info, err := s.release()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errCtxRelease, err)
	}
	events, _, err := s.events.List(ctx, tenantID, &grpcservices.EventFilters{}, s.limits.MaxEvents, 0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errCtxEvents, err)
	}
	sessions, err := s.scaci.GetSessionStatistics(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errCtxSessions, err)
	}

	entries := map[string]any{
		EntryRelease:       releaseEntry{Info: info, ServerName: s.serverName, GeneratedAt: now},
		EntryConfig:        ProjectConfig(s.cfg),
		EntryEvents:        events,
		EntrySCACISessions: sessions,
		EntryBSSCISessions: s.baseStationSessions(tenantID),
		EntrySchema:        schemaEntry{SchemaVersion: info.SchemaVersion, Version: info.Version},
	}
	archive, err := writeArchive(entries, now)
	if err != nil {
		return nil, err
	}
	if len(archive) > s.limits.MaxBundleBytes {
		return nil, fmt.Errorf(errFmtTooLarge, ErrBundleTooLarge, len(archive))
	}
	return &Bundle{Archive: archive, Filename: config.DiagnosticsArchiveFilename, ContentType: config.DiagnosticsContentType, GeneratedAt: now}, nil
}

// baseStationSessions keeps the sessions resolved to the caller's tenant; other
// tenants' base stations contribute to the count only.
func (s *Service) baseStationSessions(tenantID int64) bsSessionsEntry {
	entry := bsSessionsEntry{Sessions: []bsSessionSummary{}}
	if s.bssci == nil {
		return entry
	}
	for _, raw := range s.bssci.GetConnectedSessions() {
		entry.TotalConnected++
		if resolved, ok := raw[bssci.SessionKeyResolvedTenantID].(int64); !ok || resolved != tenantID {
			continue
		}
		if len(entry.Sessions) >= s.limits.MaxSessions {
			continue
		}
		bsEui, _ := raw[bssci.SessionKeyBaseStationEUI].(uint64)
		connected, _ := raw[bssci.SessionKeyConnected].(bool)
		handshake, _ := raw[bssci.SessionKeyHandshakeComplete].(bool)
		negotiated, _ := raw[bssci.SessionKeyNegotiatedVersion].(string)
		vendor, _ := raw[bssci.SessionKeyVendor].(string)
		model, _ := raw[bssci.SessionKeyModel].(string)
		name, _ := raw[bssci.SessionKeyName].(string)
		clientVersion, _ := raw[bssci.SessionKeyClientVersion].(string)
		entry.Sessions = append(entry.Sessions, bsSessionSummary{
			BsEui: mioty.FormatEUI64(bsEui), Connected: connected, HandshakeComplete: handshake,
			NegotiatedVersion: negotiated, Vendor: vendor, Model: model, Name: name, ClientVersion: clientVersion,
		})
	}
	return entry
}

// writeArchive serializes each entry as pretty JSON under its fixed name.
func writeArchive(entries map[string]any, now time.Time) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range Entries {
		payload, err := json.MarshalIndent(entries[name], "", "  ")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errCtxArchive, err)
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: now})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", errCtxArchive, err)
		}
		if _, err := w.Write(payload); err != nil {
			return nil, fmt.Errorf("%s: %w", errCtxArchive, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("%s: %w", errCtxArchive, err)
	}
	return buf.Bytes(), nil
}
