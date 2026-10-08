package builders

import (
	"context"

	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/adapters/streamwake"
	pkgconfig "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
)

// streamWakes wake the uplink and the event streams when the database
// announces a stored row, so the streams need not wait for their poll.
type streamWakes struct {
	uplinks *streamwake.Signal
	events  *streamwake.Signal
}

// startStreamWakes relays the database's stored-row notifications to the
// stream wakes until stop. A listener that fails leaves the streams on their
// poll: it is logged, and the service keeps running.
func startStreamWakes(ctx context.Context, storage pkgconfig.StorageConfig, log logger.Logger) (wakes streamWakes, stop func()) {
	wakes = streamWakes{uplinks: streamwake.NewSignal(), events: streamwake.NewSignal()}
	listener := postgres.NewNotificationListener(StorageOptions(storage).DSN(), map[string]func(){
		postgres.ChannelUplinkStored: wakes.uplinks.Notify,
		postgres.ChannelEventStored:  wakes.events.Notify,
	}, log)
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := listener.Run(ctx); err != nil {
			log.Error(LogStreamWakeListenerStopped, logger.Err(err))
		}
	}()
	log.Info(LogStreamWakesWired)
	return wakes, func() {
		cancel()
		<-done
	}
}
