package main

import (
	"fmt"
	"sort"
)

// Key-material classification labels for the per-surface report.
const (
	classEnvelope     = "envelope"
	classPlaintext    = "plaintext"
	classLegacy       = "legacy"
	classLegacyLocked = "legacy_locked"
	classMalformed    = "malformed"
)

// counts tracks the classification of one surface.
type counts struct {
	Envelope     int // already a valid envelope
	Plaintext    int // cleartext key material
	Legacy       int // legacy ciphertext, decryptable with the supplied key
	LegacyLocked int // legacy ciphertext, no legacy key supplied
	Malformed    int // none of the above
	Converted    int // rewritten as an envelope during apply
	Failed       int // conversion attempted and failed
}

func (c *counts) dirty() bool {
	return c.Plaintext > 0 || c.Legacy > 0 || c.LegacyLocked > 0 || c.Malformed > 0
}

// record counts one classified value and reports whether apply converts it.
func (c *counts) record(class string) bool {
	switch class {
	case classEnvelope:
		c.Envelope++
	case classPlaintext:
		c.Plaintext++
		return true
	case classLegacy:
		c.Legacy++
		return true
	case classLegacyLocked:
		c.LegacyLocked++
	default:
		c.Malformed++
	}
	return false
}

// report aggregates every surface plus the endpoint_keys reconciliation.
type report struct {
	order    []string
	surfaces map[string]*counts

	epKeysRows        int
	epKeysArchiveRows int
	epKeysResolved    int
	epKeysExported    int
	epKeysTablesGone  bool
	epKeysConflicts   []string
}

func newReport() *report {
	return &report{surfaces: map[string]*counts{}}
}

func (r *report) surface(name string) *counts {
	if c, ok := r.surfaces[name]; ok {
		return c
	}
	c := &counts{}
	r.surfaces[name] = c
	r.order = append(r.order, name)
	return c
}

// clean reports whether every surface holds only envelopes and the
// endpoint_keys subsystem is gone or empty.
func (r *report) clean() bool {
	for _, c := range r.surfaces {
		if c.dirty() {
			return false
		}
	}
	return r.epKeysRows == 0 && r.epKeysArchiveRows == 0 && len(r.epKeysConflicts) == 0
}

// applyComplete reports whether an apply run left nothing behind: no failed
// conversions, no locked legacy rows, no malformed values, no conflicts.
func (r *report) applyComplete() bool {
	for _, c := range r.surfaces {
		if c.Failed > 0 || c.LegacyLocked > 0 || c.Malformed > 0 {
			return false
		}
	}
	return len(r.epKeysConflicts) == 0
}

func (r *report) print(mode string) {
	fmt.Printf(reportFmtHeader, mode)
	names := append([]string(nil), r.order...)
	sort.Strings(names)
	for _, name := range names {
		c := r.surfaces[name]
		fmt.Printf(reportFmtSurfaceCounts,
			name, c.Envelope, c.Plaintext, c.Legacy, c.LegacyLocked, c.Malformed, c.Converted, c.Failed)
	}
	if r.epKeysTablesGone {
		fmt.Println("  endpoint_keys: tables already dropped")
	} else {
		fmt.Printf(reportFmtEndpointKeysCounts,
			r.epKeysRows, r.epKeysArchiveRows, r.epKeysResolved, r.epKeysExported, len(r.epKeysConflicts))
		for _, conflict := range r.epKeysConflicts {
			fmt.Printf(reportFmtConflict, conflict)
		}
	}
}
