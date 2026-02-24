package main

import "github.com/prometheus/client_golang/prometheus"

var (
	gamesCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "farkle_games_created_total",
			Help: "Total number of games created",
		},
	)

	gamesJoinedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "farkle_games_joined_total",
			Help: "Total number of players that joined a game",
		},
	)

	activeGames = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "farkle_active_games",
			Help: "Number of active games in memory",
		},
	)

	wsConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "farkle_ws_connections",
			Help: "Current number of active WebSocket connections",
		},
	)

	farklesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "farkle_farkles_total",
			Help: "Total number of farkle events",
		},
	)

	rollDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "farkle_roll_duration_seconds",
			Help:    "Duration of roll handling in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	prometheus.MustRegister(
		gamesCreatedTotal,
		gamesJoinedTotal,
		activeGames,
		wsConnections,
		farklesTotal,
		rollDuration,
	)
}

