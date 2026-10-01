package bssci

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/crypto"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

var errExpiryBackfillDown = errors.New("expiry backfill unavailable")

// A station registered without a certificate is pinned to the first one
// naming it, and that certificate's expiry is recorded with the pin.
func TestBindStationCertificate_PinRecordsExpiry(t *testing.T) {
	directory := &fakeBSDirectory{backfillResult: true}
	binder, claim, cert, _ := newEnforcementBinder(t, directory)
	claim.SubjectEUI = &claim.BaseStationEUI
	directory.station = RegisteredBaseStation{ID: 7, TenantID: 42}

	require.NoError(t, binder.BindStationCertificate(testutil.TestContext(), claim))
	assert.Equal(t, 1, directory.backfillCalls, "the fingerprint is pinned")
	assert.Equal(t, []time.Time{cert.NotAfter}, directory.expiries, "the pinned certificate's expiry is recorded")
}

// A station whose fingerprint was pinned before expiries were recorded gets
// the expiry of the matching certificate on its next connect.
func TestBindStationCertificate_MatchWithBlankExpiryRecordsIt(t *testing.T) {
	directory := &fakeBSDirectory{}
	binder, claim, cert, _ := newEnforcementBinder(t, directory)
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(cert.Raw),
	}

	require.NoError(t, binder.BindStationCertificate(testutil.TestContext(), claim))
	assert.Equal(t, []time.Time{cert.NotAfter}, directory.expiries)
}

// A stored expiry is never touched: the binder does not even ask to write one.
func TestBindStationCertificate_ExistingExpiryUntouched(t *testing.T) {
	directory := &fakeBSDirectory{}
	binder, claim, cert, _ := newEnforcementBinder(t, directory)
	stored := enforcementCertNotAfter.AddDate(-1, 0, 0)
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(cert.Raw),
		TLSCertExpiresAt:   &stored,
	}

	require.NoError(t, binder.BindStationCertificate(testutil.TestContext(), claim))
	assert.Empty(t, directory.expiries)
}

// A refused certificate leaves no trace: its expiry is not recorded.
func TestBindStationCertificate_MismatchRecordsNoExpiry(t *testing.T) {
	directory := &fakeBSDirectory{}
	binder, claim, _, _ := newEnforcementBinder(t, directory)
	otherCert, _ := makeEnforcementCert(t, "CA-FE-CA-FE-CA-FE-CA-FE")
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(otherCert.Raw),
	}

	require.Error(t, binder.BindStationCertificate(testutil.TestContext(), claim))
	assert.Empty(t, directory.expiries)
}

// The expiry only feeds the certificate display, so failing to record it
// never refuses a station whose certificate is bound.
func TestBindStationCertificate_ExpiryWriteFailureStillBinds(t *testing.T) {
	directory := &fakeBSDirectory{expiryErr: errExpiryBackfillDown}
	binder, claim, cert, _ := newEnforcementBinder(t, directory)
	directory.station = RegisteredBaseStation{
		ID: 7, TenantID: 42,
		TLSCertFingerprint: crypto.CertFingerprintSHA256(cert.Raw),
	}

	assert.NoError(t, binder.BindStationCertificate(testutil.TestContext(), claim))
}
