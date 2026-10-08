// Package errorgroups groups failed operations and error events per bucket.
package errorgroups

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// Bucket names accepted by ListErrorGroups.
const (
	BucketControlPlane = "control_plane"
	BucketBaseStation  = "base_station"
	BucketEndpoint     = "endpoint"
	BucketDownlink     = "downlink"
)

// ErrInvalidBucket reports a bucket outside the four known names.
var ErrInvalidBucket = errors.New("invalid error bucket")

// ErrBucketNotReadable reports a bucket holding event categories the caller's roles may not read.
var ErrBucketNotReadable = errors.New("error bucket not readable")

// ErrInvalidTimeRange reports a window whose start lies after its end.
var ErrInvalidTimeRange = errors.New("invalid time range")

const errCtxListErrorGroups = "list error groups"

// bucketSelection is the typed definition of one bucket: which event
// categories, at which severities, optionally narrowed by event type prefix,
// and whether failed SCACI operations join in.
type bucketSelection struct {
	categories        []string
	severities        []string
	eventTypePrefixes []string
	scaciFailures     bool
}

var failureSeverities = []string{models.EventSeverityError, models.EventSeverityCritical}

var buckets = map[string]bucketSelection{
	BucketControlPlane: {
		categories:    []string{models.EventCategorySCACI, models.EventCategorySession, models.EventCategorySecurity, models.EventCategorySystem, models.EventCategoryProtocol},
		severities:    failureSeverities,
		scaciFailures: true,
	},
	BucketBaseStation: {
		categories: []string{models.EventCategoryBaseStation, models.EventCategoryBSSCI},
		severities: failureSeverities,
	},
	BucketEndpoint: {
		categories: []string{models.EventCategoryEndpoint},
		severities: failureSeverities,
	},
	BucketDownlink: {
		categories:        []string{models.EventCategoryMessage},
		severities:        []string{models.EventSeverityWarning, models.EventSeverityError, models.EventSeverityCritical},
		eventTypePrefixes: []string{"dl_data_", "downlink.", "message.delivery."},
	},
}

// Reader groups failures in the event store.
type Reader interface {
	ListErrorGroups(ctx context.Context, filter models.ErrorGroupFilter) ([]*models.EventErrorGroup, int64, error)
}

// Service implements grpcservices.ErrorGroupService.
type Service struct {
	reader Reader
	clock  clock.Clock
	logger logger.Logger
}

// New creates the error-group service.
func New(reader Reader, clk clock.Clock, log logger.Logger) *Service {
	return &Service{reader: reader, clock: clk, logger: log}
}

// List returns one page of a bucket's failure groups, newest first.
func (s *Service) List(ctx context.Context, tenantID int64, bucket string, window grpcservices.ScaciWindow, limit, offset int) ([]*grpcservices.ErrorGroup, int64, error) {
	selection, ok := buckets[bucket]
	if !ok {
		return nil, 0, fmt.Errorf("%w: %s", ErrInvalidBucket, bucket)
	}
	if !authz.CanReadCategories(authz.FromContext(ctx), selection.categories) {
		return nil, 0, fmt.Errorf("%w: %s", ErrBucketNotReadable, bucket)
	}
	from, to, err := s.resolveWindow(window)
	if err != nil {
		return nil, 0, err
	}
	groups, total, err := s.reader.ListErrorGroups(ctx, models.ErrorGroupFilter{
		TenantID:             tenantID,
		Categories:           selection.categories,
		Severities:           selection.severities,
		EventTypePrefixes:    selection.eventTypePrefixes,
		IncludeSCACIFailures: selection.scaciFailures,
		From:                 from,
		To:                   to,
		Limit:                limit,
		Offset:               offset,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, LogListErrorGroupsFailed, logger.FieldTenantID, tenantID, logger.FieldError, err)
		return nil, 0, fmt.Errorf("%s: %w", errCtxListErrorGroups, err)
	}
	result := make([]*grpcservices.ErrorGroup, len(groups))
	for i, g := range groups {
		result[i] = &grpcservices.ErrorGroup{
			Bucket:     bucket,
			EventType:  g.EventType,
			Code:       g.Code,
			Message:    g.Message,
			SourceName: g.SourceName,
			FirstSeen:  g.FirstSeen,
			LastSeen:   g.LastSeen,
			Count:      g.Count,
			LastOpID:   g.LastOpID,
		}
	}
	return result, total, nil
}

// resolveWindow fills missing bounds from the default lookback and rejects
// an inverted range.
func (s *Service) resolveWindow(window grpcservices.ScaciWindow) (time.Time, time.Time, error) {
	to := s.clock.Now()
	if window.To != nil {
		to = *window.To
	}
	from := to.Add(-config.SCACIDashboardDefaultWindow)
	if window.From != nil {
		from = *window.From
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, ErrInvalidTimeRange
	}
	return from, to, nil
}

var _ grpcservices.ErrorGroupService = (*Service)(nil)
