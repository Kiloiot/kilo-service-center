package grpc

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	kcerrors "github.com/Kiloiot/kilo-service-center/KC-DB/common/errors"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/basestation"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/audit"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// BaseStationHandlers serves the base station RPCs, their metrics and the
// server-wide location listing.
type BaseStationHandlers struct {
	basestationSvc      grpcservices.BaseStationService
	statsStore          MessageStore
	statusReq           bssci.StatusRequester
	pingCmd             bssci.PingCommander
	sessionDir          bssci.SessionDirectory
	bssciSessionCloser  BSSCISessionCloser
	bsEventRecorder     basestation.EventRecorder
	audit               AuditRecorder
	availabilityReader  BaseStationAvailabilityReader
	messageBucketReader BaseStationMessageBucketReader
	orgMapper           OrgMapper
	log                 logger.Logger
}

// BaseStationHandlerDeps wires BaseStationHandlers; BaseStations, Stats,
// StatusReq, Ping and Sessions are required.
type BaseStationHandlerDeps struct {
	BaseStations   grpcservices.BaseStationService
	Stats          MessageStore
	StatusReq      bssci.StatusRequester
	Ping           bssci.PingCommander
	Sessions       bssci.SessionDirectory
	SessionCloser  BSSCISessionCloser
	EventRecorder  basestation.EventRecorder
	Availability   BaseStationAvailabilityReader
	MessageBuckets BaseStationMessageBucketReader
	Orgs           OrgMapper
}

// NewBaseStationHandlers validates the group and builds it.
func NewBaseStationHandlers(d BaseStationHandlerDeps, recorder AuditRecorder, log logger.Logger) (*BaseStationHandlers, error) {
	switch {
	case d.BaseStations == nil:
		return nil, errors.New(errMsgBasestationSvcCannotBeNil)
	case d.Stats == nil:
		return nil, errors.New(errMsgStatsStoreCannotBeNil)
	case d.StatusReq == nil:
		return nil, errors.New(errMsgStatusReqCannotBeNil)
	case d.Ping == nil:
		return nil, errors.New(errMsgPingCmdCannotBeNil)
	case d.Sessions == nil:
		return nil, errors.New(errMsgSessionDirCannotBeNil)
	}
	return &BaseStationHandlers{
		basestationSvc:      d.BaseStations,
		statsStore:          d.Stats,
		statusReq:           d.StatusReq,
		pingCmd:             d.Ping,
		sessionDir:          d.Sessions,
		bssciSessionCloser:  d.SessionCloser,
		bsEventRecorder:     d.EventRecorder,
		audit:               recorder,
		availabilityReader:  d.Availability,
		messageBucketReader: d.MessageBuckets,
		orgMapper:           d.Orgs,
		log:                 log,
	}, nil
}

// maxMetricsBuckets caps bucket count to bound handler allocation and the availability loop.
const maxMetricsBuckets = 10000

// BaseStationAvailabilityReader reads a base station's connected intervals, the
// source for time-weighted availability. Narrow consumer-side port (ISP).
type BaseStationAvailabilityReader interface {
	GetBaseStationOnlineIntervals(ctx context.Context, tenantID, baseStationID int64,
		start, end time.Time) ([]mioty.BaseStationOnlineInterval, error)
}

// BaseStationMessageBucketReader reads received-message counts grouped into
// UTC-aligned buckets. Narrow consumer-side port (ISP).
type BaseStationMessageBucketReader interface {
	CountBaseStationMessagesByBucket(ctx context.Context, tenantID int64, bsEui []byte,
		start, end time.Time, intervalSeconds int64) (map[int64]int64, error)
}

// GetBaseStationAvailability returns the per-bucket online fraction (0..1) for a
// base station over [start, end). Buckets are UTC-aligned, sorted ascending, and
// missing buckets are 0. A base station with no sessions yields all-zero buckets.
func (s *BaseStationHandlers) GetBaseStationAvailability(ctx context.Context,
	req *pb.GetBaseStationAvailabilityRequest) (*pb.GetBaseStationAvailabilityResponse, error) {

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantContextRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantContextRequired))
	}

	eui, start, end, err := validateMetricsRequest(req.BsEui, req.StartTime, req.EndTime, req.IntervalSeconds)
	if err != nil {
		return nil, err
	}

	if s.availabilityReader == nil {
		s.log.ErrorContext(ctx, LogBaseStationAvailabilityReaderNotConfigured)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetBaseStationAvailabilityFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetBaseStationAvailabilityFailed))
	}

	// A missing base station is NotFound; a valid one with no sessions yields all-zero buckets.
	bs, err := s.resolveBaseStationForMetrics(ctx, eui, tenantID, grpcerrors.ErrTokenGetBaseStationAvailabilityFailed)
	if err != nil {
		return nil, err
	}

	intervals, err := s.availabilityReader.GetBaseStationOnlineIntervals(ctx, tenantID, bs.ID, start, end)
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToGetBaseStationOnlineIntervals, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetBaseStationAvailabilityFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetBaseStationAvailabilityFailed))
	}

	availability, lastPoint := computeAvailabilityBuckets(intervals, start, end, time.Now().UTC(), req.IntervalSeconds)

	resp := &pb.GetBaseStationAvailabilityResponse{
		BsEui:           req.BsEui,
		Availability:    availability,
		IntervalSeconds: req.IntervalSeconds,
	}
	if !lastPoint.IsZero() {
		resp.LastPointTimestamp = timestamppb.New(lastPoint)
	}
	return resp, nil
}

// GetBaseStationMessagesReceived returns per-bucket received-message (RX uplink)
// counts for a base station over [start, end). Buckets are UTC-aligned, sorted
// ascending, and missing buckets are 0.
func (s *BaseStationHandlers) GetBaseStationMessagesReceived(ctx context.Context,
	req *pb.GetBaseStationMessagesReceivedRequest) (*pb.GetBaseStationMessagesReceivedResponse, error) {

	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantContextRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantContextRequired))
	}

	eui, start, end, err := validateMetricsRequest(req.BsEui, req.StartTime, req.EndTime, req.IntervalSeconds)
	if err != nil {
		return nil, err
	}

	if s.messageBucketReader == nil {
		s.log.ErrorContext(ctx, LogBaseStationMessageBucketReaderNotConfigured)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetBaseStationMessagesReceivedFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetBaseStationMessagesReceivedFailed))
	}

	// A missing base station is NotFound; a valid one with no messages yields all-zero buckets.
	if _, err := s.resolveBaseStationForMetrics(ctx, eui, tenantID, grpcerrors.ErrTokenGetBaseStationMessagesReceivedFailed); err != nil {
		return nil, err
	}

	counts, err := s.messageBucketReader.CountBaseStationMessagesByBucket(ctx, tenantID, eui[:], start, end, req.IntervalSeconds)
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToCountBaseStationMessagesByBucket, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetBaseStationMessagesReceivedFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetBaseStationMessagesReceivedFailed))
	}

	received, lastPoint := densifyMessageBuckets(counts, start, end, req.IntervalSeconds)

	resp := &pb.GetBaseStationMessagesReceivedResponse{
		BsEui:           req.BsEui,
		Received:        received,
		IntervalSeconds: req.IntervalSeconds,
	}
	if !lastPoint.IsZero() {
		resp.LastPointTimestamp = timestamppb.New(lastPoint)
	}
	return resp, nil
}

// resolveBaseStationForMetrics resolves a tenant-scoped base station for a metrics
// read. A missing base station is mapped to NotFound; any other lookup error is
// logged and mapped to the caller's failure token.
func (s *BaseStationHandlers) resolveBaseStationForMetrics(ctx context.Context, eui models.EUI, tenantID int64,
	failedToken string) (*models.BaseStation, error) {

	bs, err := s.basestationSvc.GetByEUI(ctx, eui[:], tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
		}
		s.log.ErrorContext(ctx, LogFailedToLookUpBaseStationForMetrics, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(failedToken), grpcerrors.ResolveErrorMessage(failedToken))
	}
	if bs == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	return bs, nil
}

// validateMetricsRequest validates the shared metrics request fields and returns
// the parsed EUI and the resolved [start, end) window.
func validateMetricsRequest(bsEui string, startTS, endTS *timestamppb.Timestamp,
	intervalSeconds int64) (models.EUI, time.Time, time.Time, error) {

	var zero models.EUI

	if bsEui == "" {
		return zero, time.Time{}, time.Time{}, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	eui := models.EUIFromString(bsEui)
	// Zero EUI = parse error or literal all-zero; only the former is invalid.
	if eui == zero && !isAllZeroEUI(bsEui) {
		return zero, time.Time{}, time.Time{}, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Both bucket endpoints must be non-negative Unix times so integer-division flooring is correct.
	if intervalSeconds <= 0 || startTS == nil || endTS == nil || !endTS.AsTime().After(startTS.AsTime()) ||
		startTS.AsTime().Unix() < 0 || endTS.AsTime().Unix() <= 0 {
		return zero, time.Time{}, time.Time{}, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidMetricsRequest),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidMetricsRequest))
	}

	start, end := startTS.AsTime().UTC(), endTS.AsTime().UTC()

	// Reject huge bucket counts (tiny interval x huge range) before allocating.
	if _, count := bucketRange(start, end, intervalSeconds); count > maxMetricsBuckets {
		return zero, time.Time{}, time.Time{}, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidMetricsRequest),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidMetricsRequest))
	}

	// Future end_time is allowed; buckets past now read 0.
	return eui, start, end, nil
}

func isAllZeroEUI(s string) bool {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "-", ""), ":", "")
	return len(s) == 16 && strings.Trim(s, "0") == ""
}

// bucketRange returns the first UTC-aligned bucket index (epoch/interval) and the
// number of buckets covering [start, end). Metric windows are always positive Unix
// times with a positive interval, so integer division already floors.
func bucketRange(start, end time.Time, intervalSeconds int64) (firstIdx, count int64) {
	firstIdx = start.Unix() / intervalSeconds
	lastIdx := end.Add(-time.Nanosecond).Unix() / intervalSeconds
	if lastIdx < firstIdx {
		return firstIdx, 0
	}
	return firstIdx, lastIdx - firstIdx + 1
}

// bucketStart returns the UTC start time of a bucket index.
func bucketStart(index, intervalSeconds int64) time.Time {
	return time.Unix(index*intervalSeconds, 0).UTC()
}

// densifyMessageBuckets turns a sparse bucket-index→count map into a dense,
// zero-filled array ordered by time, plus the start timestamp of the last bucket.
func densifyMessageBuckets(counts map[int64]int64, start, end time.Time, intervalSeconds int64) ([]int64, time.Time) {
	firstIdx, count := bucketRange(start, end, intervalSeconds)
	if count <= 0 {
		return []int64{}, time.Time{}
	}
	received := make([]int64, count)
	for i := int64(0); i < count; i++ {
		received[i] = counts[firstIdx+i]
	}
	return received, bucketStart(firstIdx+count-1, intervalSeconds)
}

// computeAvailabilityBuckets computes the time-weighted online fraction (0..1)
// per UTC-aligned bucket from the connected intervals. Active intervals (nil End)
// are bounded by now. Returns a dense array plus the last bucket's start timestamp.
func computeAvailabilityBuckets(intervals []mioty.BaseStationOnlineInterval, start, end, now time.Time,
	intervalSeconds int64) ([]float64, time.Time) {

	firstIdx, count := bucketRange(start, end, intervalSeconds)
	if count <= 0 {
		return []float64{}, time.Time{}
	}

	bucketWidth := time.Duration(intervalSeconds) * time.Second
	availability := make([]float64, count)
	for i := int64(0); i < count; i++ {
		bStart := bucketStart(firstIdx+i, intervalSeconds)
		bEnd := bStart.Add(bucketWidth)

		var onlineSeconds float64
		for _, iv := range intervals {
			ivEnd := now
			if iv.End != nil {
				ivEnd = *iv.End
			}
			onlineSeconds += overlapSeconds(iv.Start, ivEnd, bStart, bEnd)
		}

		fraction := onlineSeconds / float64(intervalSeconds)
		if fraction > 1 {
			fraction = 1
		}
		if fraction < 0 {
			fraction = 0
		}
		availability[i] = fraction
	}

	return availability, bucketStart(firstIdx+count-1, intervalSeconds)
}

// overlapSeconds returns the overlap in seconds between [aStart, aEnd) and [bStart, bEnd).
func overlapSeconds(aStart, aEnd, bStart, bEnd time.Time) float64 {
	s := aStart
	if bStart.After(s) {
		s = bStart
	}
	e := aEnd
	if bEnd.Before(e) {
		e = bEnd
	}
	if e.After(s) {
		return e.Sub(s).Seconds()
	}
	return 0
}

// BSSCISessionCloser closes the live BSSCI session of a base station whose
// EUI changed, reporting whether the station held one.
type BSSCISessionCloser interface {
	CloseSessionByEUI(ctx context.Context, eui uint64) bool
}

// CreateBaseStation creates a new base station
func (s *BaseStationHandlers) CreateBaseStation(ctx context.Context, req *pb.CreateBaseStationRequest) (*pb.BaseStation, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate request - nil checks BEFORE any access to req.Basestation.*
	if req.Basestation == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationRequired))
	}
	if req.Basestation.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	s.log.InfoContext(ctx, grpcerrors.LogBaseStationCreating, logger.FieldEui, req.Basestation.BsEui, logger.FieldTenantIDSnake, tenantID)

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.Basestation.TenantId != "" && req.Basestation.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	eui := models.EUIFromString(req.Basestation.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	tagsJSON, err := encodeBaseStationTags(req.Basestation.Tags)
	if err != nil {
		return nil, s.internalFailure(ctx, LogFailedToEncodeBaseStationTags, err)
	}

	// Extract coordinate wrappers (nil = absent, non-nil = explicit value including 0.0)
	var lat, lon, alt *float64
	if req.Basestation.Latitude != nil {
		v := req.Basestation.Latitude.GetValue()
		lat = &v
	}
	if req.Basestation.Longitude != nil {
		v := req.Basestation.Longitude.GetValue()
		lon = &v
	}
	if req.Basestation.Altitude != nil {
		v := req.Basestation.Altitude.GetValue()
		alt = &v
	}

	// Validate lat/lon pair: both present or both absent
	if (lat != nil) != (lon != nil) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLatLonPairRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLatLonPairRequired))
	}
	if lat != nil && !models.CoordinatesInRange(*lat, *lon) {
		return nil, locationOutOfRangeError()
	}

	var desc *string
	if req.Basestation.Description != "" {
		desc = &req.Basestation.Description
	}

	// Set location source and timestamp when coordinates are provided
	var locationSource *string
	var locationUpdatedAt *time.Time
	if lat != nil && lon != nil {
		src := models.LocationSourceManual
		locationSource = &src
		now := time.Now()
		locationUpdatedAt = &now
	}

	// Convert proto to canonical model using authenticated tenant ID
	baseStation := &models.BaseStation{
		EUI:               eui,
		TenantID:          tenantID,
		Name:              req.Basestation.Name,
		Description:       desc,
		Latitude:          lat,
		Longitude:         lon,
		Altitude:          alt,
		LocationSource:    locationSource,
		LocationUpdatedAt: locationUpdatedAt,
		Tags:              tagsJSON,
	}

	// Create base station in storage
	created, err := s.basestationSvc.Create(ctx, baseStation)
	if err != nil {
		// Duplicate EUI must surface as AlreadyExists (409), not a generic Internal.
		if errors.Is(err, kcerrors.ErrDuplicate) {
			s.log.WarnContext(ctx, grpcerrors.LogBaseStationAlreadyExists, logger.FieldEui, req.Basestation.BsEui, logger.FieldTenantIDSnake, tenantID)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationEUIExists),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationEUIExists))
		}
		s.log.ErrorContext(ctx, LogFailedToCreateBaseStation, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenCreateBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenCreateBaseStationFailed))
	}

	createdEui := created.EUI.String()
	s.audit.Record(ctx, audit.Event{
		TenantID:      tenantID,
		Category:      models.EventCategoryBaseStation,
		EventType:     models.EventTypeBSRegistered,
		Title:         models.EventTitleBSRegistered,
		Description:   fmt.Sprintf(models.EventDescriptionBSRegistered, createdEui),
		SourceType:    models.SourceTypeBaseStation,
		SourceName:    createdEui,
		BaseStationID: &created.ID,
		Details:       map[string]any{bssci.EventKeyBsEui: createdEui},
	})

	return s.baseStationResponse(ctx, created)
}

// GetBaseStation retrieves a base station by EUI
func (s *BaseStationHandlers) GetBaseStation(ctx context.Context, req *pb.GetBaseStationRequest) (*pb.BaseStation, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, LogGettingBaseStation, logger.FieldEui, req.BsEui, logger.FieldTenantIDSnake, tenantID)

	// Validate request
	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	eui := models.EUIFromString(req.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Get base station from storage using authenticated tenant ID
	baseStation, err := s.basestationSvc.GetByEUI(ctx, eui[:], tenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToGetBaseStation, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetBaseStationFailed))
	}

	return s.baseStationResponse(ctx, baseStation)
}

// UpdateBaseStation updates a base station
func (s *BaseStationHandlers) UpdateBaseStation(ctx context.Context, req *pb.UpdateBaseStationRequest) (*pb.BaseStation, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if req.Basestation == nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationRequired))
	}
	if req.Basestation.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	s.log.InfoContext(ctx, LogUpdatingBaseStation, logger.FieldEui, req.Basestation.BsEui, logger.FieldTenantIDSnake, tenantID)

	// Validate EUI format
	eui := models.EUIFromString(req.Basestation.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.Basestation.TenantId != "" && req.Basestation.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Fetch existing base station to merge fields
	existing, err := s.basestationSvc.GetByEUI(ctx, eui[:], tenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToFetchBaseStationForUpdate, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateBaseStationFailed))
	}

	// FieldMask is required for partial updates
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, status.Error(
			grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateMaskRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateMaskRequired),
		)
	}

	// Start with existing values, then overlay based on mask
	baseStation := existing

	// Name
	if fieldInMask(mask, bsFieldMaskName) {
		if req.Basestation.Name != "" {
			baseStation.Name = req.Basestation.Name
		}
	}

	// Description
	if fieldInMask(mask, bsFieldMaskDescription) {
		if req.Basestation.Description != "" {
			desc := req.Basestation.Description
			baseStation.Description = &desc
		} else {
			baseStation.Description = nil
		}
	}

	// Tags
	if len(req.Basestation.Tags) > 0 {
		tagsJSON, err := encodeBaseStationTags(req.Basestation.Tags)
		if err != nil {
			return nil, s.internalFailure(ctx, LogFailedToEncodeBaseStationTags, err)
		}
		baseStation.Tags = tagsJSON
	}

	// Coordinates with FieldMask support
	latMasked := fieldInMask(mask, bsFieldMaskLatitude)
	lonMasked := fieldInMask(mask, bsFieldMaskLongitude)
	altMasked := fieldInMask(mask, bsFieldMaskAltitude)

	// If one of lat/lon is masked, both must be
	if latMasked != lonMasked {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLatLonPairRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLatLonPairRequired))
	}

	if latMasked {
		if req.Basestation.Latitude != nil {
			v := req.Basestation.Latitude.GetValue()
			baseStation.Latitude = &v
		} else {
			baseStation.Latitude = nil
		}
	}
	if lonMasked {
		if req.Basestation.Longitude != nil {
			v := req.Basestation.Longitude.GetValue()
			baseStation.Longitude = &v
		} else {
			baseStation.Longitude = nil
		}
	}
	if altMasked {
		if req.Basestation.Altitude != nil {
			v := req.Basestation.Altitude.GetValue()
			baseStation.Altitude = &v
		} else {
			baseStation.Altitude = nil
		}
	}

	// Update location source/timestamp
	if latMasked && lonMasked {
		if baseStation.Latitude != nil && baseStation.Longitude != nil {
			src := models.LocationSourceManual
			baseStation.LocationSource = &src
			now := time.Now()
			baseStation.LocationUpdatedAt = &now
		} else {
			baseStation.LocationSource = nil
			baseStation.LocationUpdatedAt = nil
		}
	}

	if baseStation.Latitude != nil && baseStation.Longitude != nil &&
		!models.CoordinatesInRange(*baseStation.Latitude, *baseStation.Longitude) {
		return nil, locationOutOfRangeError()
	}

	// Update base station in storage
	updated, err := s.basestationSvc.Update(ctx, baseStation)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToUpdateBaseStation, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateBaseStationFailed))
	}

	updatedEui := updated.EUI.String()
	s.audit.Record(ctx, audit.Event{
		TenantID:      tenantID,
		Category:      models.EventCategoryBaseStation,
		EventType:     models.EventTypeBSUpdated,
		Title:         models.EventTitleBSUpdated,
		Description:   fmt.Sprintf(models.EventDescriptionBSUpdated, updatedEui),
		SourceType:    models.SourceTypeBaseStation,
		SourceName:    updatedEui,
		BaseStationID: &updated.ID,
		Details:       map[string]any{bssci.EventKeyBsEui: updatedEui},
	})

	return s.baseStationResponse(ctx, updated)
}

// UpdateBaseStationEui updates the EUI of a base station with cascade to all dependent tables.
func (s *BaseStationHandlers) UpdateBaseStationEui(ctx context.Context, req *pb.UpdateBaseStationEuiRequest) (*pb.BaseStation, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, LogUpdatingBaseStationEUI, logger.FieldOldEui, req.BsEui, logger.FieldNewEui, req.NewBsEui, logger.FieldTenantIDSnake, tenantID)

	oldEui, newEui, err := parseEUIChange(req)
	if err != nil {
		return nil, err
	}

	// Update EUI in storage with transactional cascade
	updated, err := s.basestationSvc.UpdateEUI(ctx, tenantID, oldEui[:], newEui[:])
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if errors.Is(err, storage.ErrAlreadyExists) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationEUIExists),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationEUIExists))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToUpdateBaseStationEUI, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenUpdateBaseStationEUIFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenUpdateBaseStationEUIFailed))
	}

	s.retireOldIdentitySession(ctx, oldEui, newEui)

	return s.baseStationResponse(ctx, updated)
}

// parseEUIChange reads the station's current and new EUI, each required and
// accepted dashed or not.
func parseEUIChange(req *pb.UpdateBaseStationEuiRequest) (models.EUI, models.EUI, error) {
	if req.BsEui == "" {
		return models.EUI{}, models.EUI{}, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}
	if req.NewBsEui == "" {
		return models.EUI{}, models.EUI{}, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenNewBaseStationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenNewBaseStationEUIRequired))
	}
	oldEui := models.EUIFromString(req.BsEui)
	newEui := models.EUIFromString(req.NewBsEui)
	if oldEui == (models.EUI{}) || newEui == (models.EUI{}) {
		return models.EUI{}, models.EUI{}, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}
	return oldEui, newEui, nil
}

// retireOldIdentitySession closes the live BSSCI session held under the
// station's old EUI, since the station must reconnect with its new identity.
// Only a session actually closed takes the station offline, and that event is
// recorded here under the new EUI: the close retires the session, so its
// connection teardown records nothing.
func (s *BaseStationHandlers) retireOldIdentitySession(ctx context.Context, oldEui, newEui models.EUI) {
	if s.bssciSessionCloser == nil || !s.bssciSessionCloser.CloseSessionByEUI(ctx, oldEui.ToUint64()) {
		return
	}
	closedAt := time.Now()
	if s.bsEventRecorder == nil {
		return
	}
	offline := map[string]interface{}{
		models.EventDetailKeyIsOnline: false,
		models.EventDetailKeyReason:   reasonEUIChanged,
	}
	if err := s.bsEventRecorder.RecordEvent(ctx, newEui, models.EventTypeBaseStationOffline, closedAt, offline); err != nil {
		s.log.WarnContext(ctx, LogFailedToRecordOfflineEvent, logger.FieldError, err)
	}
}

// DeleteBaseStation deletes a base station
func (s *BaseStationHandlers) DeleteBaseStation(ctx context.Context, req *pb.DeleteBaseStationRequest) (*emptypb.Empty, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, LogDeletingBaseStation, logger.FieldEui, req.BsEui, logger.FieldTenantIDSnake, tenantID)

	// Validate request
	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	eui := models.EUIFromString(req.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	removed, err := s.basestationSvc.Delete(ctx, eui[:], tenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToDeleteBaseStation, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenDeleteBaseStationFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenDeleteBaseStationFailed))
	}

	removedEui := removed.EUI.String()
	s.audit.Record(ctx, audit.Event{
		TenantID:      tenantID,
		Category:      models.EventCategoryBaseStation,
		EventType:     models.EventTypeBSDeregistered,
		Title:         models.EventTitleBSDeregistered,
		Description:   fmt.Sprintf(models.EventDescriptionBSDeleted, removedEui),
		SourceType:    models.SourceTypeBaseStation,
		SourceName:    removedEui,
		BaseStationID: &removed.ID,
		Details: map[string]any{
			bssci.EventKeyBsEui:                  removedEui,
			models.EventDetailKeyBaseStationName: removed.Name,
		},
	})

	return &emptypb.Empty{}, nil
}

// ListBaseStations lists base stations for a tenant
func (s *BaseStationHandlers) ListBaseStations(ctx context.Context, req *pb.ListBaseStationsRequest) (*pb.ListBaseStationsResponse, error) {
	// Get authenticated tenant ID from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, LogListingBaseStations, logger.FieldTenantIDSnake, tenantID, logger.FieldPageSize, req.PageSize, logger.FieldPageToken, req.PageToken)

	// Validate tenant ID in request matches authenticated tenant (if provided)
	if req.TenantId != "" && req.TenantId != strconv.FormatInt(tenantID, 10) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantAccessDenied),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantAccessDenied))
	}

	// Default page size
	pageSize := clampHighVolumePageSize(req.PageSize, DefaultHighVolumePageSize)

	// Parse page token (simple offset-based pagination)
	offset := 0
	if req.PageToken != "" {
		_, err := fmt.Sscanf(req.PageToken, "%d", &offset)
		if err != nil {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidPageToken),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidPageToken))
		}
	}

	// List base stations from storage using authenticated tenant ID
	baseStations, err := s.basestationSvc.List(ctx, tenantID, int(pageSize), offset)
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToListBaseStations, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenListBaseStationsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenListBaseStationsFailed))
	}

	// Convert to proto
	pbBaseStations := make([]*pb.BaseStation, len(baseStations))
	for i, baseStation := range baseStations {
		pbBaseStation, err := s.baseStationResponse(ctx, baseStation)
		if err != nil {
			return nil, err
		}
		pbBaseStations[i] = pbBaseStation
	}

	// Generate next page token
	nextPageToken := ""
	if len(baseStations) == int(pageSize) {
		nextPageToken = fmt.Sprintf("%d", offset+int(pageSize))
	}

	return &pb.ListBaseStationsResponse{
		Basestations:  pbBaseStations,
		NextPageToken: nextPageToken,
	}, nil
}

// GetBaseStationStats retrieves aggregated message statistics for a base station.
func (s *BaseStationHandlers) GetBaseStationStats(ctx context.Context, req *pb.GetBaseStationStatsRequest) (*pb.GetBaseStationStatsResponse, error) {
	// Get tenant ID from context (required by fail-closed org resolver interceptor)
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenTenantContextRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenTenantContextRequired))
	}

	s.log.InfoContext(ctx, LogGettingBaseStationStats, logger.FieldBsEuiSnake, req.BsEui, logger.FieldTenantIDSnake, tenantID)

	// Validate request
	if req.BsEui == "" {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBasestationEUIRequired),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBasestationEUIRequired))
	}

	eui := models.EUIFromString(req.BsEui)
	if eui == (models.EUI{}) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidBasestationEUIFormat),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInvalidBasestationEUIFormat))
	}

	// Get message stats from storage
	stats, err := s.statsStore.GetBaseStationMessageStats(ctx, tenantID, eui[:],
		timestampToTime(req.StartTime), timestampToTime(req.EndTime))
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToGetBaseStationStats, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetBaseStationStatsFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetBaseStationStatsFailed))
	}

	// Get endpoint counts
	endpointCounts, err := s.statsStore.GetBaseStationEndpointCounts(ctx, tenantID, eui[:],
		timestampToTime(req.StartTime), timestampToTime(req.EndTime))
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToGetEndpointCounts, logger.FieldError, err)
		// Don't fail the entire request, just log the error
		endpointCounts = make(map[string]int64)
	}

	// Get last seen timestamp
	lastSeen, err := s.statsStore.GetBaseStationLastSeen(ctx, tenantID, eui[:])
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToGetLastSeenTimestamp, logger.FieldError, err)
	}

	// Get base station status from storage
	bs, err := s.basestationSvc.GetByEUI(ctx, eui[:], tenantID)
	var bsStatus string
	if err == nil && bs != nil {
		if bs.IsOnline {
			bsStatus = grpcerrors.StatusOnline
		} else {
			bsStatus = grpcerrors.StatusOffline
		}
	} else {
		bsStatus = grpcerrors.StatusOffline
	}

	// Convert to protobuf response
	response := &pb.GetBaseStationStatsResponse{
		BsEui:                 req.BsEui,
		TotalMessages:         stats.TotalMessages,
		TotalEndpoints:        stats.TotalEndpoints,
		MessagesToday:         stats.MessagesToday,
		MessagesThisWeek:      stats.MessagesThisWeek,
		MessagesThisMonth:     stats.MessagesThisMonth,
		AvgRssi:               stats.AvgRSSI,
		AvgSnr:                stats.AvgSNR,
		EndpointMessageCounts: endpointCounts,
		Status:                bsStatus,
	}

	// Add timestamps if available
	if stats.LastMessageAt != nil {
		response.LastMessageAt = timestamppb.New(*stats.LastMessageAt)
	}
	if lastSeen != nil {
		response.LastSeenAt = timestamppb.New(*lastSeen)
	}

	return response, nil
}

// RequestBaseStationStatus sends a status request to a base station
func (s *BaseStationHandlers) RequestBaseStationStatus(ctx context.Context, req *pb.BaseStationStatusRequest) (*pb.BaseStationStatusResponse, error) {
	// Extract tenant from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bsEui, err := resolveBaseStationEUI(req.BsEuiHex, req.BsEui) //nolint:staticcheck // deprecated field read for wire compatibility with legacy clients
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, LogRequestBaseStationStatusRequest,
		logger.FieldBsEuiSnake, bsEui,
		logger.FieldTenantIDSnake, tenantID)

	bsEuiArr := mioty.EUI64(bsEui).ToBytes()
	bsEuiBytes := bsEuiArr[:]

	// Verify tenant ownership first
	bs, err := s.basestationSvc.GetByEUI(ctx, bsEuiBytes, tenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToVerifyBaseStationOwnership, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationOwnershipFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationOwnershipFailed))
	}

	// Find connected session
	sessionIface := s.sessionDir.GetSessionByEUI(bsEui)
	if sessionIface == nil {
		s.log.WarnContext(ctx, grpcerrors.LogBaseStationNotConnected,
			logger.FieldBsEuiSnake, bsEui,
			logger.FieldBsName, bs.Name)
		return &pb.BaseStationStatusResponse{
			Success: false,
			Message: grpcerrors.LogBaseStationNotConnected,
			OpId:    0,
		}, nil
	}

	// Send status request - reuse existing logic
	opId, err := s.statusReq.SendStatusRequest(sessionIface)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogStatusRequestFailed,
			logger.FieldBsEuiSnake, bsEui,
			logger.FieldError, err)
		return &pb.BaseStationStatusResponse{
			Success: false,
			Message: fmt.Sprintf(grpcerrors.MsgStatusRequestFailed, err),
			OpId:    0,
		}, nil
	}

	s.log.InfoContext(ctx, grpcerrors.MsgStatusRequestSent,
		logger.FieldBsEuiSnake, bsEui,
		logger.FieldOpIDSnake, opId,
		logger.FieldTenantIDSnake, tenantID)

	return &pb.BaseStationStatusResponse{
		Success: true,
		Message: grpcerrors.MsgStatusRequestSent,
		OpId:    opId,
	}, nil
}

// InitiatePing sends a ping request to a base station (BSSCI §5.4)
func (s *BaseStationHandlers) InitiatePing(ctx context.Context, req *pb.InitiatePingRequest) (*pb.InitiatePingResponse, error) {
	// Extract tenant from context
	tenantID, err := GetTenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	bsEui, err := resolveBaseStationEUI(req.BsEuiHex, req.BsEui) //nolint:staticcheck // deprecated field read for wire compatibility with legacy clients
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, LogInitiatePingRequest,
		logger.FieldBsEuiSnake, bsEui,
		logger.FieldTenantIDSnake, tenantID)

	bsEuiArr := mioty.EUI64(bsEui).ToBytes()
	bsEuiBytes := bsEuiArr[:]

	// Verify tenant ownership
	_, err = s.basestationSvc.GetByEUI(ctx, bsEuiBytes, tenantID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
	}
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToVerifyBaseStationOwnership, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationOwnershipFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationOwnershipFailed))
	}

	// Call BSSCI server InitiatePing (defense-in-depth tenant validation)
	opId, err := s.pingCmd.InitiatePing(ctx, bsEui, tenantID)
	if err != nil {
		s.log.ErrorContext(ctx, grpcerrors.LogPingInitiateFailed,
			logger.FieldBsEuiSnake, bsEui,
			logger.FieldError, err)

		// Branch on CatalogError token for proper gRPC status codes
		var catalogErr *bssci.CatalogError
		if errors.As(err, &catalogErr) {
			switch catalogErr.Token {
			case bssci.ErrTokenSessionNotFound:
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBSSessionNotFound),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBSSessionNotFound))
			case bssci.ErrTokenCannotSendPing:
				return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenHandshakeIncomplete),
					grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenHandshakeIncomplete))
			}
		}
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	s.log.InfoContext(ctx, grpcerrors.MsgPingRequestSent,
		logger.FieldBsEuiSnake, bsEui,
		logger.FieldOpIDSnake, opId,
		logger.FieldTenantIDSnake, tenantID)

	return &pb.InitiatePingResponse{
		Success: true,
		Message: grpcerrors.MsgPingRequestSent,
		OpId:    opId,
	}, nil
}

// ListAllBaseStationLocations returns locations of all base stations across all tenants.
// Restricted to server admins (user.IsAdmin=true).
func (s *BaseStationHandlers) ListAllBaseStationLocations(ctx context.Context, _ *pb.ListAllBaseStationLocationsRequest) (*pb.ListAllBaseStationLocationsResponse, error) {
	// Fetch all base stations with coordinates
	stations, err := s.basestationSvc.ListAllLocations(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, LogFailedToListAllBaseStationLocations, logger.FieldError, err)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	// Collect unique tenant IDs and batch-resolve to org UUIDs
	tenantOrgMap := make(map[int64]string)
	for _, bs := range stations {
		if _, ok := tenantOrgMap[bs.TenantID]; !ok {
			tenantOrgMap[bs.TenantID] = ""
		}
	}

	if s.orgMapper == nil {
		s.log.ErrorContext(ctx, LogOrgMapperNotConfigured)
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
	}

	for tid := range tenantOrgMap {
		orgID, mapErr := s.orgMapper.GetDefaultOrgForTenant(ctx, tid)
		if mapErr != nil {
			s.log.ErrorContext(ctx, LogFailedToMapTenantToOrg, logger.FieldTenantIDSnake, tid, logger.FieldError, mapErr)
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInternalError),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInternalError))
		}
		tenantOrgMap[tid] = orgID
	}

	// Build response
	locations := make([]*pb.BaseStationLocation, 0, len(stations))
	for _, bs := range stations {
		loc := &pb.BaseStationLocation{
			BsEui:    mioty.FormatEUIBytes(bs.EUI[:]),
			Name:     bs.Name,
			IsOnline: bs.IsOnline,
			OrgId:    tenantOrgMap[bs.TenantID],
		}

		if bs.Latitude != nil {
			loc.Latitude = *bs.Latitude
		}
		if bs.Longitude != nil {
			loc.Longitude = *bs.Longitude
		}
		if bs.Altitude != nil {
			loc.Altitude = wrapperspb.Double(*bs.Altitude)
		}
		if bs.LocationSource != nil {
			loc.LocationSource = *bs.LocationSource
		}

		locations = append(locations, loc)
	}

	return &pb.ListAllBaseStationLocationsResponse{
		Locations:  locations,
		TotalCount: int32(len(locations)), //nolint:gosec // location count is bounded by DB query, no overflow risk
	}, nil
}

func locationOutOfRangeError() error {
	return status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenLocationOutOfRange),
		grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenLocationOutOfRange))
}
