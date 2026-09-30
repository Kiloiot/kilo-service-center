package certificates

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// keyStation names the base station whose private key is taken.
type keyStation struct {
	id  int64
	eui []byte
}

// takePrivateKey hands out a base station's stored private key once. The
// stored copy is cleared only after it decrypts, matches want when want is
// set, and its download is recorded, so a key that cannot be served is never
// lost; every refusal, including a key already taken, is ErrNotFound, one
// that could not be decrypted is also ErrKeyUnreadable, and one whose
// download could not be recorded is also ErrKeyDownloadNotRecorded.
func (s *Service) takePrivateKey(ctx context.Context, tenantID int64, station keyStation, want []byte) ([]byte, error) {
	var key []byte
	err := s.bsRepo.TakeTLSKey(ctx, tenantID, station.eui, func(sealed string) error {
		decrypted, err := s.keyEncryptor.DecryptKey(sealed)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrKeyUnreadable, err)
		}
		if want != nil && !bytes.Equal(decrypted, want) {
			return ErrKeySuperseded
		}
		if err := s.disclosures.RecordRequired(ctx, privateKeyDownloadedEvent(tenantID, station)); err != nil {
			return fmt.Errorf("%w: %w", ErrKeyDownloadNotRecorded, err)
		}
		key = decrypted
		return nil
	})
	if err != nil {
		s.logger.ErrorContext(ctx, LogDownloadCertFailed, logger.FieldBsEuiSnake, mioty.FormatEUIBytes(station.eui), logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	return key, nil
}

// privateKeyDownloadedEvent records who took the station's private key; never the key.
func privateKeyDownloadedEvent(tenantID int64, station keyStation) audit.Event {
	bsEui := mioty.FormatEUIBytes(station.eui)
	stationID := station.id
	return audit.Event{
		TenantID:      tenantID,
		EventType:     models.EventTypeCertificatePrivateKeyDownloaded,
		Title:         models.EventTitleCertificatePrivateKeyDownloaded,
		Description:   fmt.Sprintf(models.EventDescriptionCertificatePrivateKeyDownloadedFmt, bsEui),
		SourceType:    models.SourceTypeBaseStation,
		SourceName:    bsEui,
		BaseStationID: &stationID,
		Details:       map[string]any{models.EventDetailKeyBsEui: bsEui},
	}
}
