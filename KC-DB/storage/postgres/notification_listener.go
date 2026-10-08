package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// Channels the triggers of migrations 000182 and 000183 notify, with an
// empty payload, when a row is stored.
const (
	// ChannelUplinkStored announces a stored uplink (a ulData row of messages).
	ChannelUplinkStored = "kc_uplink_stored"
	// ChannelEventStored announces a stored system event, or one moved forward in time.
	ChannelEventStored = "kc_event_stored"
)

// NotificationListener relays the notifications of its channels to their
// handlers. Its connection reconnects on its own with a doubling backoff;
// after a reconnect every handler runs once, since the notifications sent
// while it was down are lost.
type NotificationListener struct {
	dsn      string
	handlers map[string]func()
	log      logger.Logger
}

// NewNotificationListener returns a listener on the database dsn names that
// runs handlers[channel] for every notification on channel.
func NewNotificationListener(dsn string, handlers map[string]func(), log logger.Logger) *NotificationListener {
	return &NotificationListener{dsn: dsn, handlers: handlers, log: log}
}

// Run relays notifications until ctx ends; it fails only when the server
// refuses to listen on a channel.
func (l *NotificationListener) Run(ctx context.Context) error {
	listener := pq.NewListener(l.dsn, listenerReconnectFirst, listenerReconnectCap, l.logEvent)
	closeOnDone := context.AfterFunc(ctx, func() { l.close(listener) })
	defer func() {
		if closeOnDone() {
			l.close(listener)
		}
	}()
	for channel := range l.handlers {
		// Listen waits for a connection; closing the listener when ctx ends releases it.
		if err := listener.Listen(channel); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf(errFmtListenChannel, channel, err)
		}
	}
	l.relay(ctx, listener)
	return nil
}

func (l *NotificationListener) relay(ctx context.Context, listener *pq.Listener) {
	ping := time.NewTicker(listenerPingInterval)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case notification, open := <-listener.Notify:
			if !open {
				return
			}
			l.dispatch(notification)
		case <-ping.C:
			// Ping waits for the connection's reply, which queues behind the notifications this loop drains.
			go l.ping(listener)
		}
	}
}

// dispatch runs the handler of the notification's channel; the nil
// notification that follows a reconnect runs every handler.
func (l *NotificationListener) dispatch(notification *pq.Notification) {
	if notification == nil {
		for _, handle := range l.handlers {
			handle()
		}
		return
	}
	if handle, ok := l.handlers[notification.Channel]; ok {
		handle()
	}
}

func (l *NotificationListener) logEvent(event pq.ListenerEventType, err error) {
	if err != nil {
		l.log.Warn(logMsgNotificationListenerLost, logger.FieldEvent, event.String(), logger.FieldError, err)
		return
	}
	l.log.Info(logMsgNotificationListenerConnected, logger.FieldEvent, event.String())
}

func (l *NotificationListener) ping(listener *pq.Listener) {
	if err := listener.Ping(); err != nil {
		l.log.Warn(logMsgNotificationListenerPingFailed, logger.FieldError, err)
	}
}

func (l *NotificationListener) close(listener *pq.Listener) {
	if err := listener.Close(); err != nil {
		l.log.Warn(logMsgNotificationListenerCloseFailed, logger.FieldError, err)
	}
}
