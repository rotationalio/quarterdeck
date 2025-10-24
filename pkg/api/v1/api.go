package api

import "context"

//===========================================================================
// Service Interface
//===========================================================================

// Client defines the service interface for interacting with the Quarterdeck
// internal API (e.g. the API that users can integrate with).
type Client interface {
	Status(context.Context) (*StatusReply, error)
	DBInfo(context.Context) (*DBInfo, error)
}

//===========================================================================
// Top Level Requests and Responses
//===========================================================================

// Reply contains standard fields that are used for generic API responses and errors.
type Reply struct {
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
	ErrorDetail ErrorDetail `json:"errors,omitempty"`
}

// PageQuery is used for paginated queries.
type PageQuery struct {
	PageSize      int    `json:"page_size,omitempty" url:"page_size,omitempty" form:"page_size"`
	NextPageToken string `json:"next_page_token,omitempty" url:"next_page_token,omitempty" form:"next_page_token"`
}

type Page struct {
	PrevPageToken string `json:"prev_page_token,omitempty"`
	NextPageToken string `json:"next_page_token,omitempty"`
	PageSize      int    `json:"page_size,omitempty"`
}

// Returned on status requests.
type StatusReply struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime,omitempty"`
	Version string `json:"version,omitempty"`
}

// A copy of the sql.DBStats struct that implements JSON serialization.
type DBInfo struct {
	MaxOpenConnections int    `json:"max_open_connections"` // Maximum number of open connections to the database.
	OpenConnections    int    `json:"open_connections"`     // The number of established connections both in use and idle.
	InUse              int    `json:"in_use"`               // The number of connections currently in use.
	Idle               int    `json:"idle"`                 // The number of idle connections.
	WaitCount          int64  `json:"wait_count"`           // The total number of connections waited for.
	WaitDuration       string `json:"wait_duration"`        // The total time blocked waiting for a new connection.
	MaxIdleClosed      int64  `json:"max_idle_closed"`      // The total number of connections closed due to SetMaxIdleConns.
	MaxIdleTimeClosed  int64  `json:"max_idle_time_closed"` // The total number of connections closed due to SetConnMaxIdleTime.
	MaxLifetimeClosed  int64  `json:"max_lifetime_closed"`  // The total number of connections closed due to SetConnMaxLifetime.
}
