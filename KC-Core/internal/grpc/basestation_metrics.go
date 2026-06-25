package grpc

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

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

// Compile-time verification that postgres.DB satisfies the metrics readers.
var (
	_ BaseStationAvailabilityReader  = (*postgres.DB)(nil)
	_ BaseStationMessageBucketReader = (*postgres.DB)(nil)
)

// WithBaseStationMetricsReaders wires the optional base-station metrics readers.
func (s *CoreService) WithBaseStationMetricsReaders(availability BaseStationAvailabilityReader,
	messages BaseStationMessageBucketReader) *CoreService {
	s.availabilityReader = availability
	s.messageBucketReader = messages
	return s
}

// GetBaseStationAvailability returns the per-bucket online fraction (0..1) for a
// base station over [start, end). Buckets are UTC-aligned, sorted ascending, and
// missing buckets are 0. A base station with no sessions yields all-zero buckets.
func (s *CoreService) GetBaseStationAvailability(ctx context.Context,
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
		s.log.ErrorContext(ctx, "Base station availability reader not configured")
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
		s.log.ErrorContext(ctx, "Failed to get base station online intervals", "error", err)
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
func (s *CoreService) GetBaseStationMessagesReceived(ctx context.Context,
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
		s.log.ErrorContext(ctx, "Base station message bucket reader not configured")
		return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenGetBaseStationMessagesReceivedFailed),
			grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenGetBaseStationMessagesReceivedFailed))
	}

	// A missing base station is NotFound; a valid one with no messages yields all-zero buckets.
	if _, err := s.resolveBaseStationForMetrics(ctx, eui, tenantID, grpcerrors.ErrTokenGetBaseStationMessagesReceivedFailed); err != nil {
		return nil, err
	}

	counts, err := s.messageBucketReader.CountBaseStationMessagesByBucket(ctx, tenantID, eui[:], start, end, req.IntervalSeconds)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to count base station messages by bucket", "error", err)
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
func (s *CoreService) resolveBaseStationForMetrics(ctx context.Context, eui models.EUI, tenantID int64,
	failedToken string) (*models.BaseStation, error) {

	bs, err := s.basestationSvc.GetByEUI(ctx, eui[:], tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(grpcerrors.GetGRPCCode(grpcerrors.ErrTokenBaseStationNotFound),
				grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenBaseStationNotFound))
		}
		s.log.ErrorContext(ctx, "Failed to look up base station for metrics", "error", err)
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
	if eui == zero {
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

	return eui, startTS.AsTime().UTC(), endTS.AsTime().UTC(), nil
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
