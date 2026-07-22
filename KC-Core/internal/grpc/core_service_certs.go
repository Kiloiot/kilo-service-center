// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// Certificate handlers

// GenerateCertificate generates a new certificate for a base station.
func (s *CoreService) GenerateCertificate(ctx context.Context, req *pb.GenerateCertificateRequest) (*pb.GenerateCertificateResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	// Get tenant ID for certificate persistence (optional - persistence skipped if unavailable)
	tenantID, _ := GetTenantFromContext(ctx)

	certReq := &grpcservices.CertificateRequest{
		BsEUI:           req.BsEui,
		BaseStationName: req.BaseStationName,
		ValidityDays:    req.ValidityDays,
		TenantID:        tenantID,
	}

	resp, err := s.certSvc.GenerateCertificate(ctx, certReq)
	if err != nil {
		s.log.ErrorContext(ctx, "generate certificate failed", "bs_eui", req.BsEui, "error", err)

		// Detect missing server CA and return an actionable error
		errStr := err.Error()
		if strings.Contains(errStr, grpcerrors.ErrTokenCACertReadFailed) ||
			strings.Contains(errStr, grpcerrors.ErrTokenCAKeyReadFailed) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCACertReadFailed),
				grpcerrors.MsgServerCertRequiredBeforeBS)
		}

		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertGenerationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertGenerationFailed))
	}

	// Emit audit event for certificate generation.
	if s.eventWriter != nil {
		detailsMap := map[string]interface{}{
			bssci.EventKeyBsEui: req.BsEui,
			"validityDays":      req.ValidityDays,
		}
		if req.BaseStationName != "" {
			detailsMap["baseStationName"] = req.BaseStationName
		}
		detailsJSON, _ := json.Marshal(detailsMap)
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			TenantID:    strconv.FormatInt(tenantID, 10),
			EventType:   models.EventTypeCertificateGenerated,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleCertificateGenerated,
			Description: fmt.Sprintf(models.EventDescriptionCertificateGeneratedFmt, req.BsEui),
			SourceType:  models.SourceTypeBaseStation,
			SourceName:  req.BsEui,
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	pbResp := &pb.GenerateCertificateResponse{
		BsEui:            resp.BsEUI,
		ServiceCenterUrl: resp.ServiceCenterURL,
		DownloadUrls:     resp.DownloadURLs,
	}
	if resp.ExpiresAt != nil {
		pbResp.ExpiresAt = timestamppb.New(*resp.ExpiresAt)
	}
	return pbResp, nil
}

// DownloadCertificate downloads a certificate by ID and type.
func (s *CoreService) DownloadCertificate(ctx context.Context, req *pb.DownloadCertificateRequest) (*pb.DownloadCertificateResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	// Validate cert_type
	if req.CertType == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertTypeRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertTypeRequired))
	}
	if req.CertType != grpcerrors.CertTypeCA && req.CertType != grpcerrors.CertTypeClient && req.CertType != grpcerrors.CertTypeKey {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertTypeRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertTypeRequired))
	}

	// Validate id (required for generated certificate downloads)
	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	// Download from temp directory by ID
	data, filename, err := s.certSvc.DownloadCertificateByID(ctx, req.CertType, req.Id)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogDownloadCertFailed, "cert_type", req.CertType, "cert_id", req.Id, "error", err)
		// Preserve service-level error tokens (don't mask invalid cert_type)
		if strings.Contains(err.Error(), grpcerrors.ErrTokenCertTypeRequired) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertTypeRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertTypeRequired))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertNotFound))
	}

	return &pb.DownloadCertificateResponse{
		Content:     data,
		Filename:    filename,
		ContentType: grpcerrors.ContentTypePEM,
	}, nil
}

// GenerateServerCertificates generates new server certificates.
func (s *CoreService) GenerateServerCertificates(ctx context.Context, _ *pb.GenerateServerCertificatesRequest) (*pb.GenerateServerCertificatesResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if err := s.certSvc.GenerateServerCertificates(ctx); err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogGenerateServerCertsFailed, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertServerGenerationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertServerGenerationFailed))
	}

	// Emit audit event for server certificate generation.
	if s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"scope": "server",
		})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			EventType:   models.EventTypeCertificateServerGenerated,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleCertificateServerGenerated,
			Description: fmt.Sprintf(models.EventDescriptionCertificateServerGeneratedFmt, "kc-core"),
			SourceType:  models.SourceTypeSystem,
			SourceName:  "kc-core",
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return &pb.GenerateServerCertificatesResponse{
		Success: true,
		Message: grpcerrors.MsgCertsGenerated,
	}, nil
}

// RenewServerCertificates renews server certificates.
func (s *CoreService) RenewServerCertificates(ctx context.Context, _ *pb.RenewServerCertificatesRequest) (*pb.RenewServerCertificatesResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if err := s.certSvc.RenewServerCertificates(ctx); err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogRenewServerCertsFailed, "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertRenewalFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertRenewalFailed))
	}

	// Emit audit event for server certificate renewal.
	if s.eventWriter != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"scope": "server",
		})
		_ = s.eventWriter.CreateEvent(ctx, &models.SystemEvent{
			EventType:   models.EventTypeCertificateServerRenewed,
			Category:    models.EventCategoryAudit,
			Severity:    models.EventSeverityInfo,
			Title:       models.EventTitleCertificateServerRenewed,
			Description: fmt.Sprintf(models.EventDescriptionCertificateServerRenewedFmt, "kc-core"),
			SourceType:  models.SourceTypeSystem,
			SourceName:  "kc-core",
			Details:     detailsJSON,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	return &pb.RenewServerCertificatesResponse{
		Success: true,
		Message: grpcerrors.MsgCertsRenewed,
	}, nil
}

// GetServerCertificateStatus returns the status of server certificates.
func (s *CoreService) GetServerCertificateStatus(ctx context.Context, _ *pb.GetServerCertificateStatusRequest) (*pb.GetServerCertificateStatusResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	certStatus, err := s.certSvc.GetServerCertificateStatus(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "get server certificate status failed", "error", err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertStatusFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertStatusFailed))
	}

	resp := &pb.GetServerCertificateStatusResponse{}

	// Build server cert status
	if certStatus.HasServerCert {
		serverCert := &pb.CertificateStatus{
			Subject: certStatus.Subject,
			Issuer:  certStatus.Issuer,
			IsValid: !certStatus.NeedsRenewal,
		}
		if certStatus.ServerCertExpiry != nil {
			serverCert.NotAfter = timestamppb.New(*certStatus.ServerCertExpiry)
			daysUntil := int32(time.Until(*certStatus.ServerCertExpiry).Hours() / 24)
			serverCert.DaysUntilExpiry = daysUntil
		}
		resp.ServerCert = serverCert
	}

	// Build CA cert status
	if certStatus.HasCACert {
		caCert := &pb.CertificateStatus{
			IsValid: true,
		}
		if certStatus.CACertExpiry != nil {
			caCert.NotAfter = timestamppb.New(*certStatus.CACertExpiry)
		}
		resp.CaCert = caCert
	}

	return resp, nil
}

// DownloadBaseStationCertificate downloads stored TLS certificates from base station record.
func (s *CoreService) DownloadBaseStationCertificate(ctx context.Context, req *pb.DownloadBaseStationCertificateRequest) (*pb.DownloadCertificateResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}
	if req.CertType == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertTypeRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertTypeRequired))
	}

	bsEui, err := parseEUI(req.BsEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	data, filename, err := s.certSvc.GetStoredCertificate(ctx, tenantID, bsEui, req.CertType)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogDownloadCertFailed, "cert_type", req.CertType, "bs_eui", req.BsEui, "error", err)
		if strings.Contains(err.Error(), grpcerrors.ErrTokenServiceNotConfigured) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
		}
		if strings.Contains(err.Error(), grpcerrors.ErrTokenCertTypeRequired) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertTypeRequired),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertTypeRequired))
		}
		if strings.Contains(err.Error(), grpcerrors.ErrTokenBaseStationNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertNotFound))
	}

	return &pb.DownloadCertificateResponse{
		Content:     data,
		Filename:    filename,
		ContentType: grpcerrors.ContentTypePEM,
	}, nil
}
