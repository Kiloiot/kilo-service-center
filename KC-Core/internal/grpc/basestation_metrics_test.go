package grpc

import (
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const testBucketSeconds = int64(3600) // 1 hour

func testWindow() (start, end time.Time) {
	start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end = start.Add(3 * time.Hour)
	return start, end
}

func TestDensifyMessageBuckets_DenseSortedZeroFilled(t *testing.T) {
	start, end := testWindow()
	firstIdx, count := bucketRange(start, end, testBucketSeconds)
	if count != 3 {
		t.Fatalf("expected 3 buckets, got %d", count)
	}

	// Only the first and last buckets have data; the middle must be zero-filled.
	counts := map[int64]int64{firstIdx: 5, firstIdx + 2: 7}

	received, lastPoint := densifyMessageBuckets(counts, start, end, testBucketSeconds)

	want := []int64{5, 0, 7}
	if len(received) != len(want) {
		t.Fatalf("length: got %d want %d", len(received), len(want))
	}
	for i := range want {
		if received[i] != want[i] {
			t.Errorf("bucket %d: got %d want %d", i, received[i], want[i])
		}
	}
	if !lastPoint.Equal(bucketStart(firstIdx+2, testBucketSeconds)) {
		t.Errorf("lastPoint mismatch: got %v", lastPoint)
	}
}

func TestDensifyMessageBuckets_EmptyRange(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	received, lastPoint := densifyMessageBuckets(nil, start, start, testBucketSeconds)
	if received == nil || len(received) != 0 {
		t.Errorf("expected non-nil empty slice, got %#v", received)
	}
	if !lastPoint.IsZero() {
		t.Errorf("expected zero lastPoint, got %v", lastPoint)
	}
}

func TestComputeAvailabilityBuckets_TimeWeightedAndClamped(t *testing.T) {
	start, end := testWindow()
	now := end

	// Bucket 0 fully online; bucket 1 online for the first half; bucket 2 offline.
	endB0B1Half := start.Add(90 * time.Minute)
	intervals := []mioty.BaseStationOnlineInterval{
		{Start: start, End: &endB0B1Half},
	}

	availability, lastPoint := computeAvailabilityBuckets(intervals, start, end, now, testBucketSeconds)

	want := []float64{1.0, 0.5, 0.0}
	if len(availability) != len(want) {
		t.Fatalf("length: got %d want %d", len(availability), len(want))
	}
	for i := range want {
		if availability[i] != want[i] {
			t.Errorf("bucket %d: got %v want %v", i, availability[i], want[i])
		}
	}
	firstIdx, count := bucketRange(start, end, testBucketSeconds)
	if !lastPoint.Equal(bucketStart(firstIdx+count-1, testBucketSeconds)) {
		t.Errorf("lastPoint mismatch: got %v", lastPoint)
	}
}

func TestComputeAvailabilityBuckets_ActiveSessionBoundedByNow(t *testing.T) {
	start, end := testWindow()
	now := start.Add(2 * time.Hour) // active "now" sits at the start of bucket 2

	// Active session (nil End) started at window start; online up to now only.
	intervals := []mioty.BaseStationOnlineInterval{{Start: start}}

	availability, _ := computeAvailabilityBuckets(intervals, start, end, now, testBucketSeconds)

	want := []float64{1.0, 1.0, 0.0}
	for i := range want {
		if availability[i] != want[i] {
			t.Errorf("bucket %d: got %v want %v", i, availability[i], want[i])
		}
	}
}

func TestComputeAvailabilityBuckets_OverlapClampedToOne(t *testing.T) {
	start, end := testWindow()
	now := end

	// Two overlapping intervals in bucket 0 must not exceed 1.0.
	e0 := start.Add(time.Hour)
	e1 := start.Add(time.Hour)
	intervals := []mioty.BaseStationOnlineInterval{
		{Start: start, End: &e0},
		{Start: start, End: &e1},
	}

	availability, _ := computeAvailabilityBuckets(intervals, start, end, now, testBucketSeconds)
	if availability[0] != 1.0 {
		t.Errorf("expected clamp to 1.0, got %v", availability[0])
	}
}
