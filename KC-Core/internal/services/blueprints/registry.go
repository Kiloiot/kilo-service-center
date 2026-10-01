package blueprints

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"

	blueprintregistry "github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/blueprints"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	blueprintconstants "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/blueprint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/google/uuid"
)

// SubmitToRegistry submits a blueprint to an external registry.
func (s *Service) SubmitToRegistry(ctx context.Context, id uuid.UUID, req *grpcservices.RegistrySubmitRequest) (*grpcservices.RegistrySubmitResult, error) {
	tenantID, err := s.tenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Domain validation: Check if registry is enabled
	if s.registryCfg == nil || !s.registryCfg.Enabled {
		return nil, ErrRegistryDisabled
	}

	// Domain validation: api_url must be configured when enabled
	if s.registryCfg.APIURL == "" {
		return nil, ErrRegistryAPIURLRequired
	}

	// Get blueprint to verify it exists and check submission status
	bp, err := s.blueprintRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrBlueprintNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetBlueprint, err)
	}

	// Domain validation: Check if already submitted (RegistryPRURL is *string)
	if bp.RegistryPRURL != nil && *bp.RegistryPRURL != "" {
		return nil, ErrAlreadySubmitted
	}

	// Resolve device model and manufacturer for structured directory naming
	deviceModel, err := s.deviceModelRepo.GetByID(ctx, tenantID, bp.DeviceModelID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrDeviceModelNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetDeviceModel, err)
	}

	manufacturer, err := s.manufacturerRepo.GetByID(ctx, tenantID, deviceModel.ManufacturerID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrManufacturerNotFound
		}
		return nil, fmt.Errorf("%s: %w", errOpGetManufacturer, err)
	}

	// Build structured path segments
	mfrSlug, err := blueprintconstants.PathSegment(manufacturer.Name)
	if err != nil {
		return nil, err
	}

	// Prefer model code (URL-friendly slug), fall back to name
	var modelSlug string
	if deviceModel.Code != "" {
		modelSlug, err = blueprintconstants.ModelCodeSegment(deviceModel.Code)
	} else {
		modelSlug, err = blueprintconstants.PathSegment(deviceModel.Name)
	}
	if err != nil {
		return nil, err
	}

	normalizedVer := blueprintconstants.NormalizeVersion(bp.Version)
	versionSlug, err := blueprintconstants.VersionSegment(normalizedVer)
	if err != nil {
		return nil, err
	}

	// Generate structured file path: {manufacturer}/{model}/{version}.json
	filePath := fmt.Sprintf("%s%s/%s/%s%s", s.registryCfg.BlueprintPath, mfrSlug, modelSlug, versionSlug, s.registryCfg.FileExtension)

	// Generate branch name with short ID suffix for uniqueness
	shortID := id.String()[:8]
	branchName := fmt.Sprintf("%s%s/%s/%s/%s", s.registryCfg.BranchPrefix, mfrSlug, modelSlug, versionSlug, shortID)

	// Create registry client
	client, err := blueprintregistry.NewRegistryClient(s.registryCfg, s.logger)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpCreateRegistryClient, err)
	}

	// Preflight: check if file already exists in the base branch
	exists, err := client.FileExists(ctx, s.registryCfg.BaseBranch, filePath)
	if err != nil {
		s.logger.ErrorContext(ctx, LogRegistryFileCheckFailed, logger.FieldPath, filePath, logger.FieldError, err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrRegistryAPIError, err)
	}
	if exists {
		return nil, ErrRegistryVersionAlreadyExists
	}

	// Get base branch SHA
	baseSHA, err := client.BaseBranchSHA(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, LogRegistryBaseSHAFailed, logger.FieldError, err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrBranchCreateFailed, err)
	}

	// Create branch
	if err := client.CreateBranch(ctx, baseSHA, branchName); err != nil {
		s.logger.ErrorContext(ctx, LogRegistryBranchCreateFailed, logger.FieldBranch, branchName, logger.FieldError, err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrBranchCreateFailed, err)
	}

	// Prepare blueprint content with enriched metadata
	blueprintContent, err := s.prepareBlueprintContent(bp, manufacturer.Name, deviceModel.Name, deviceModel.Code)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpPrepareContent, err)
	}

	authorName := req.ContributorName
	authorEmail := req.ContributorEmail

	// Commit the blueprint file using format constants
	commitMessage := fmt.Sprintf(blueprintconstants.RegistryCommitMsgFmt, manufacturer.Name, deviceModel.Name, normalizedVer)
	if req.Notes != "" {
		commitMessage = fmt.Sprintf("%s\n\n%s", commitMessage, req.Notes)
	}

	commitSHA, err := client.CreateOrUpdateFile(ctx, branchName, filePath, blueprintContent, commitMessage, authorName, authorEmail)
	if err != nil {
		s.logger.ErrorContext(ctx, LogRegistryCommitFailed, logger.FieldPath, filePath, logger.FieldError, err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrCommitFailed, err)
	}

	// Create submission request (pull request)
	prTitle := fmt.Sprintf(blueprintconstants.RegistryPRTitleFmt, manufacturer.Name, deviceModel.Name, normalizedVer)
	prBody := fmt.Sprintf(blueprintconstants.RegistryPRBodyFmt,
		manufacturer.Name, deviceModel.Name, normalizedVer,
		mioty.FormatEUIBytes(bp.TypeEUI), authorName, authorEmail, req.Notes)

	prURL, err := client.CreateSubmissionRequest(ctx, prTitle, prBody, branchName, s.registryCfg.BaseBranch)
	if err != nil {
		s.logger.ErrorContext(ctx, LogRegistrySubmissionCreateFailed, logger.FieldError, err)
		if isSentinelError(err) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrSubmissionCreateFailed, err)
	}

	// Update blueprint with registry PR URL
	repoPath := fmt.Sprintf("%s/%s", s.registryCfg.Owner, s.registryCfg.Repo)
	if err := s.blueprintRepo.UpdateRegistryInfo(ctx, tenantID, bp.IsSystem, id, repoPath, commitSHA, prURL, false); err != nil {
		s.logger.ErrorContext(ctx, LogRegistryInfoUpdateFailed, logger.FieldID, id, logger.FieldError, err)
		// Don't fail the operation - PR was created successfully
	}

	s.logger.InfoContext(
		ctx, LogBlueprintSubmitted,
		logger.FieldID, id,
		logger.FieldPrURL, prURL,
		logger.FieldCommitSha, commitSHA,
		logger.FieldBranch, branchName,
		logger.FieldFilePath, filePath,
	)

	return &grpcservices.RegistrySubmitResult{
		PRUrl:      prURL,
		CommitSHA:  commitSHA,
		BranchName: branchName,
	}, nil
}

// isSentinelError checks if an error is a registry sentinel that should be propagated directly.
func isSentinelError(err error) bool {
	return errors.Is(err, ErrRegistryAuthFailed) ||
		errors.Is(err, ErrRegistryPermissionDenied) ||
		errors.Is(err, ErrRegistryRateLimited) ||
		errors.Is(err, ErrRegistryAPIError) ||
		errors.Is(err, ErrRegistryVersionAlreadyExists)
}
