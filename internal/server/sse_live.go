package server

import "github.com/alexandrmotologa/claimguard/internal/telemetry"

// Broadcaster re-exports the telemetry broadcaster for server routing.
type Broadcaster = telemetry.Broadcaster
type LiveEvent = telemetry.LiveEvent

// NewBroadcaster creates a new live event broadcaster.
var NewBroadcaster = telemetry.NewBroadcaster
