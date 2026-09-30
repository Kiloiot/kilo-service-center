package certificates

import (
	"context"
	"fmt"
	"os"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// GetStoredCertificate serves a certificate file of a base station the tenant
// owns: the CA is the service center CA every station certificate is issued
// with, whether or not issuance stored a copy on the station; the client
// certificate and the private key are the copies issuance stored.
func (s *Service) GetStoredCertificate(ctx context.Context, tenantID int64, bsEui []byte, certType string) ([]byte, string, error) {
	if certType != CertTypeCA && certType != CertTypeClient && certType != CertTypeKey {
		s.logger.ErrorContext(ctx, LogDownloadCertFailed, logger.FieldCertType, certType, logger.FieldError, ErrTypeRequired)
		return nil, "", ErrTypeRequired
	}

	bs, err := s.bsRepo.GetByEUI(ctx, tenantID, bsEui)
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertBSNotFound, logger.FieldBsEuiSnake, mioty.FormatEUIBytes(bsEui), logger.FieldError, err)
		return nil, "", fmt.Errorf("%w: %w", ErrBaseStationNotFound, err)
	}

	euiHex := mioty.FormatEUIBytes(bsEui)
	switch certType {
	case CertTypeCA:
		ca, err := s.serviceCA(ctx)
		if err != nil {
			return nil, "", err
		}
		return ca, fmt.Sprintf(downloadNameFmtBSCACert, euiHex), nil
	case CertTypeClient:
		if bs.TLSCertificate == nil || *bs.TLSCertificate == "" {
			return nil, "", ErrCertificateNotStored
		}
		return []byte(*bs.TLSCertificate), fmt.Sprintf(downloadNameFmtClientCert, euiHex), nil
	default:
		key, err := s.takePrivateKey(ctx, tenantID, keyStation{id: bs.ID, eui: bsEui}, nil)
		if err != nil {
			return nil, "", err
		}
		return key, fmt.Sprintf(downloadNameFmtPrivateKey, euiHex), nil
	}
}

// serviceCA reads the service center CA certificate.
func (s *Service) serviceCA(ctx context.Context) ([]byte, error) {
	data, err := os.ReadFile(s.caCertPath()) // #nosec G304 - configured certificate directory and a fixed file name
	if err != nil {
		s.logger.ErrorContext(ctx, LogCertCACertReadFailed, logger.FieldError, err)
		return nil, fmt.Errorf("%w: %w", ErrCACertRead, err)
	}
	return data, nil
}
