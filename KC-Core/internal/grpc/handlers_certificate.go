// Package grpc provides gRPC service implementations.
package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/certificates"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/common/validation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// CertificateHandlers serves the certificate RPCs.
type CertificateHandlers struct {
	certSvc          grpcservices.CertificateService
	audit            AuditRecorder
	platformTenantID int64
	log              logger.Logger
}

// CertificateHandlerDeps wires CertificateHandlers; a nil service answers
// its RPCs as not configured. Server certificate events are filed under
// PlatformTenantID, the tenant that owns server-level events.
type CertificateHandlerDeps struct {
	Certificates     grpcservices.CertificateService
	PlatformTenantID int64
}

// NewCertificateHandlers builds the group.
func NewCertificateHandlers(d CertificateHandlerDeps, recorder AuditRecorder, log logger.Logger) *CertificateHandlers {
	return &CertificateHandlers{
		certSvc:          d.Certificates,
		audit:            recorder,
		platformTenantID: d.PlatformTenantID,
		log:              log,
	}
}

// Certificate handlers

const sourceNameKCCore = "kc-core"

// GenerateCertificate generates a new certificate for a base station.
// sourceNameKCCore identifies KC-Core as a system event source.
func (s *CertificateHandlers) GenerateCertificate(ctx context.Context, req *pb.GenerateCertificateRequest) (*pb.GenerateCertificateResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	// Tenant context is mandatory: certificate issuance is scoped to the tenant
	// that owns the base station, and issuing without a tenant would allow
	// minting a certificate for any EUI.
	tenantID, tenantErr := GetTenantFromContext(ctx)
	if tenantErr != nil || tenantID <= 0 {
		s.log.WarnContext(ctx, grpcerrors.LogCertIssuanceRequiresTenant, logger.FieldBsEuiSnake, req.BsEui, logger.FieldError, tenantErr)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingTenantCtx),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenMissingTenantCtx))
	}

	certReq := &grpcservices.CertificateRequest{
		BsEUI:           req.BsEui,
		BaseStationName: req.BaseStationName,
		ValidityDays:    req.ValidityDays,
		TenantID:        tenantID,
	}

	resp, err := s.certSvc.GenerateCertificate(ctx, certReq)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogGenerateCertificateFailed, logger.FieldBsEuiSnake, req.BsEui, logger.FieldError, err)

		// Domain sentinels survive the service boundary: match with errors.Is,
		// never on error text. A missing server CA gets the actionable setup
		// message; everything else resolves through the catalog token.
		if errors.Is(err, certificates.ErrCACertRead) || errors.Is(err, certificates.ErrCAKeyRead) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCACertReadFailed),
				grpcerrors.MsgServerCertRequiredBeforeBS)
		}
		token := certErrorToken(err, grpcerrors.ErrTokenCertGenerationFailed)
		return nil, status.Error(grpcerrors.GetGRPCCode(token),
			grpcerrors.ResolveErrorMessage(token))
	}

	bsEui := models.EUIFromString(resp.BsEUI).String()
	details := map[string]any{
		bssci.EventKeyBsEui: bsEui,
		"validityDays":      req.ValidityDays,
	}
	if req.BaseStationName != "" {
		details[models.EventDetailKeyBaseStationName] = req.BaseStationName
	}
	s.audit.Record(ctx, audit.Event{
		TenantID:      tenantID,
		Category:      models.EventCategoryBaseStation,
		EventType:     models.EventTypeCertificateGenerated,
		Title:         models.EventTitleCertificateGenerated,
		Description:   fmt.Sprintf(models.EventDescriptionCertificateGeneratedFmt, bsEui),
		SourceType:    models.SourceTypeBaseStation,
		SourceName:    bsEui,
		BaseStationID: &resp.BaseStationID,
		Details:       details,
	})

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

// DownloadCertificate downloads one file of a generated bundle; only the
// tenant owning the bundle's base station can read it.
func (s *CertificateHandlers) DownloadCertificate(ctx context.Context, req *pb.DownloadCertificateRequest) (*pb.DownloadCertificateResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate cert_type
	if req.CertType == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertTypeRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertTypeRequired))
	}
	if req.CertType != certificates.CertTypeCA && req.CertType != certificates.CertTypeClient && req.CertType != certificates.CertTypeKey {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertTypeRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertTypeRequired))
	}

	// Validate id (required for generated certificate downloads)
	if req.Id == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenIDRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenIDRequired))
	}

	data, filename, err := s.certSvc.DownloadCertificateByID(ctx, tenantID, req.CertType, req.Id)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogDownloadCertFailed, logger.FieldCertType, req.CertType, logger.FieldCertIDSnake, req.Id, logger.FieldError, err)
		token := certErrorToken(err, grpcerrors.ErrTokenCertNotFound)
		return nil, status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
	}

	return &pb.DownloadCertificateResponse{
		Content:     data,
		Filename:    filename,
		ContentType: grpcerrors.ContentTypePEM,
	}, nil
}

// GenerateServerCertificates generates new server certificates.
func (s *CertificateHandlers) GenerateServerCertificates(ctx context.Context, _ *pb.GenerateServerCertificatesRequest) (*pb.GenerateServerCertificatesResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if err := s.certSvc.GenerateServerCertificates(ctx); err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogGenerateServerCertsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertServerGenerationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertServerGenerationFailed))
	}

	s.audit.Record(ctx, audit.Event{
		TenantID:    s.platformTenantID,
		EventType:   models.EventTypeCertificateServerGenerated,
		Title:       models.EventTitleCertificateServerGenerated,
		Description: fmt.Sprintf(models.EventDescriptionCertificateServerGeneratedFmt, sourceNameKCCore),
		SourceType:  models.SourceTypeSystem,
		SourceName:  sourceNameKCCore,
		Details:     map[string]any{"scope": certificates.CertTypeServer},
	})

	return &pb.GenerateServerCertificatesResponse{
		Success: true,
		Message: grpcerrors.MsgCertsGenerated,
	}, nil
}

// RenewServerCertificates renews server certificates.
func (s *CertificateHandlers) RenewServerCertificates(ctx context.Context, _ *pb.RenewServerCertificatesRequest) (*pb.RenewServerCertificatesResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	if err := s.certSvc.RenewServerCertificates(ctx); err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogRenewServerCertsFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertRenewalFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertRenewalFailed))
	}

	s.audit.Record(ctx, audit.Event{
		TenantID:    s.platformTenantID,
		EventType:   models.EventTypeCertificateServerRenewed,
		Title:       models.EventTitleCertificateServerRenewed,
		Description: fmt.Sprintf(models.EventDescriptionCertificateServerRenewedFmt, sourceNameKCCore),
		SourceType:  models.SourceTypeSystem,
		SourceName:  sourceNameKCCore,
		Details:     map[string]any{"scope": certificates.CertTypeServer},
	})

	return &pb.RenewServerCertificatesResponse{
		Success: true,
		Message: grpcerrors.MsgCertsRenewed,
	}, nil
}

// GetServerCertificateStatus returns the status of server certificates.
func (s *CertificateHandlers) GetServerCertificateStatus(ctx context.Context, _ *pb.GetServerCertificateStatusRequest) (*pb.GetServerCertificateStatusResponse, error) {
	if s.certSvc == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenServiceNotConfigured),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenServiceNotConfigured))
	}

	certStatus, err := s.certSvc.GetServerCertificateStatus(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogGetServerCertStatusFailed, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCertStatusFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCertStatusFailed))
	}

	return &pb.GetServerCertificateStatusResponse{
		ServerCert:   certificateStatusToProto(certStatus.Server),
		CaCert:       certificateStatusToProto(certStatus.CA),
		RenewalNames: certStatus.RenewalNames,
	}, nil
}

func certificateStatusToProto(info *grpcservices.CertificateInfo) *pb.CertificateStatus {
	if info == nil {
		return nil
	}
	return &pb.CertificateStatus{
		Subject:         info.Subject,
		Issuer:          info.Issuer,
		NotBefore:       timestamppb.New(info.NotBefore),
		NotAfter:        timestamppb.New(info.NotAfter),
		DaysUntilExpiry: info.DaysUntilExpiry,
		IsValid:         info.Valid,
	}
}

// DownloadBaseStationCertificate downloads stored TLS certificates from base station record.
func (s *CertificateHandlers) DownloadBaseStationCertificate(ctx context.Context, req *pb.DownloadBaseStationCertificateRequest) (*pb.DownloadCertificateResponse, error) {
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

	bsEui, err := validation.ParseEUIBytes(req.BsEui)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	data, filename, err := s.certSvc.GetStoredCertificate(ctx, tenantID, bsEui, req.CertType)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogDownloadCertFailed, logger.FieldCertType, req.CertType, logger.FieldBsEuiSnake, req.BsEui, logger.FieldError, err)
		token := certErrorToken(err, grpcerrors.ErrTokenCertNotFound)
		return nil, status.Error(grpcerrors.GetGRPCCode(token), grpcerrors.ResolveErrorMessage(token))
	}

	return &pb.DownloadCertificateResponse{
		Content:     data,
		Filename:    filename,
		ContentType: grpcerrors.ContentTypePEM,
	}, nil
}

// certErrorToken maps a certificates domain sentinel onto its catalog token,
// or fallback for an unrecognized error.
func certErrorToken(err error, fallback string) string {
	for _, pair := range certSentinelTokens {
		if errors.Is(err, pair.sentinel) {
			return pair.token
		}
	}
	return fallback
}

// certSentinelTokens pairs every certificates domain sentinel with its
// catalog token; a sentinel that wraps others comes before them.
var certSentinelTokens = []struct {
	sentinel error
	token    string
}{
	{certificates.ErrPersistenceFailed, grpcerrors.ErrTokenCertPersistenceFailed},
	{certificates.ErrGeneratorNotFound, grpcerrors.ErrTokenCertGeneratorNotFound},
	{certificates.ErrServiceNotConfigured, grpcerrors.ErrTokenServiceNotConfigured},
	{certificates.ErrInvalidBaseStationEUI, grpcerrors.ErrTokenInvalidBasestationEUIFormat},
	{certificates.ErrInvalidValidityPeriod, grpcerrors.ErrTokenInvalidValidityPeriod},
	{certificates.ErrTenantRequired, grpcerrors.ErrTokenMissingTenantCtx},
	{certificates.ErrBaseStationNotFound, grpcerrors.ErrTokenBaseStationNotFound},
	{certificates.ErrDirectoryCreate, grpcerrors.ErrTokenCertDirCreateFailed},
	{certificates.ErrCACertRead, grpcerrors.ErrTokenCACertReadFailed},
	{certificates.ErrCACertCopy, grpcerrors.ErrTokenCACertCopyFailed},
	{certificates.ErrCAKeyRead, grpcerrors.ErrTokenCAKeyReadFailed},
	{certificates.ErrCAKeyCopy, grpcerrors.ErrTokenCAKeyCopyFailed},
	{certificates.ErrGenerationFailed, grpcerrors.ErrTokenCertGenerationFailed},
	{certificates.ErrTypeRequired, grpcerrors.ErrTokenCertTypeRequired},
	{certificates.ErrKeyUnreadable, grpcerrors.ErrTokenInternalError},
	{certificates.ErrKeyDownloadNotRecorded, grpcerrors.ErrTokenInternalError},
	{certificates.ErrNotFound, grpcerrors.ErrTokenCertNotFound},
	{certificates.ErrCertificateNotStored, grpcerrors.ErrTokenCertNotStored},
	{certificates.ErrServerGenerationFailed, grpcerrors.ErrTokenCertServerGenerationFailed},
	{certificates.ErrNoCertificatesToRenew, grpcerrors.ErrTokenNoCertsToRenew},
}
