package certificates

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// DownloadCertificateByID serves one file of a generated bundle to the tenant
// that owns the bundle's base station; an unknown bundle, a malformed id and a
// base station of another tenant all read as ErrNotFound.
func (s *Service) DownloadCertificateByID(ctx context.Context, tenantID int64, certType, certID string) ([]byte, string, error) {
	var filename, downloadNameFmt string
	switch certType {
	case CertTypeCA:
		filename, downloadNameFmt = caCertFileName, downloadNameCACert
	case CertTypeClient:
		filename, downloadNameFmt = clientCertFileName, downloadNameFmtClientCert
	case CertTypeKey:
		filename, downloadNameFmt = clientKeyFileName, downloadNameFmtPrivateKey
	default:
		s.logger.ErrorContext(ctx, LogDownloadCertFailed, logger.FieldCertType, certType, logger.FieldError, ErrTypeRequired)
		return nil, "", ErrTypeRequired
	}

	b, err := s.ownedBundle(ctx, tenantID, certID)
	if err != nil {
		return nil, "", err
	}

	path := filepath.Join(b.dir, filename)
	data, err := os.ReadFile(path) // #nosec G304 - canonical UUID bundle directory and a fixed file name
	if err != nil {
		return nil, "", fmt.Errorf("%w: %w", ErrNotFound, err)
	}

	if certType == CertTypeKey {
		if data, err = s.takeBundleKey(ctx, tenantID, b, path, data); err != nil {
			return nil, "", err
		}
	}

	if certType == CertTypeCA {
		return data, downloadNameFmt, nil
	}
	return data, fmt.Sprintf(downloadNameFmt, mioty.FormatEUIBytes(b.station.eui)), nil
}

// takeBundleKey serves a bundle's private key only by taking the station's
// stored copy, which issuance wrote alongside it: the row lock lets exactly
// one download win, and a bundle whose key was taken or superseded is
// refused. The bundle's own file is removed once it is served or can never
// be served again; a transient failure keeps it for a retry.
func (s *Service) takeBundleKey(ctx context.Context, tenantID int64, b certBundle, path string, fileKey []byte) ([]byte, error) {
	key, err := s.takePrivateKey(ctx, tenantID, b.station, fileKey)
	if err != nil {
		if bundleKeyRetired(err) {
			s.removeBundleKey(ctx, path)
		}
		return nil, err
	}
	s.removeBundleKey(ctx, path)
	return key, nil
}

// bundleKeyRetired reports a refusal after which the bundle's key can never
// be served: the station's stored key was already taken or is a newer one.
func bundleKeyRetired(err error) bool {
	return errors.Is(err, storage.ErrNotFound) || errors.Is(err, ErrKeySuperseded)
}

func (s *Service) removeBundleKey(ctx context.Context, path string) {
	if err := os.Remove(path); err != nil {
		s.logger.WarnContext(ctx, LogCertBundleKeyRemoveFailed, logger.FieldPath, path, logger.FieldError, err)
	}
}

// certBundle is a generated certificate bundle and the base station it was issued for.
type certBundle struct {
	dir     string
	station keyStation
}

// ownedBundle resolves certID to its bundle directory and verifies that the
// bundle's base station belongs to tenantID.
func (s *Service) ownedBundle(ctx context.Context, tenantID int64, certID string) (certBundle, error) {
	id, err := uuid.Parse(certID)
	if err != nil {
		return certBundle{}, fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	dir := filepath.Join(s.settings.tempDir, id.String())

	infoData, err := os.ReadFile(filepath.Join(dir, certInfoFileName)) // #nosec G304 - canonical UUID bundle directory
	if err != nil {
		return certBundle{}, fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	var info struct {
		BsEui string `json:"bsEui"`
	}
	if err := json.Unmarshal(infoData, &info); err != nil {
		s.logger.ErrorContext(ctx, LogCertUnmarshalFailed, logger.FieldError, err)
		return certBundle{}, fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	eui, err := validation.ParseEUI(info.BsEui)
	if err != nil {
		return certBundle{}, fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	euiBytes := binary.BigEndian.AppendUint64(nil, eui)
	bs, err := s.bsRepo.GetByEUI(ctx, tenantID, euiBytes)
	if err != nil {
		return certBundle{}, fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	return certBundle{dir: dir, station: keyStation{id: bs.ID, eui: euiBytes}}, nil
}

// CleanupExpiredCertificates removes the bundle directories whose download
// window had passed at now.
func (s *Service) CleanupExpiredCertificates(ctx context.Context, now time.Time) {
	entries, err := os.ReadDir(s.settings.tempDir)
	if err != nil {
		return
	}
	cutoffTime := now.Add(-certDownloadWindow)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.ModTime().Before(cutoffTime) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(s.settings.tempDir, entry.Name())); err != nil {
			s.logger.WarnContext(ctx, LogCertExpiredDirRemoveFailed,
				logger.FieldDir, entry.Name(), logger.FieldError, err)
		}
	}
}
