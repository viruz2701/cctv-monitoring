package models

import "time"

type P2PDevice struct {
	ID           int       `db:"id"`
	Serial       string    `db:"serial"`
	Brand        string    `db:"brand"` // hikvision, dahua, xiongmai, reolink, ilnk
	SecurityCode string    `db:"security_code"`
	CloudUser    *string   `db:"cloud_user"` // optional
	CloudPass    *string   `db:"cloud_pass"` // encrypted
	Status       string    `db:"status"`     // online, offline
	LastSeen     time.Time `db:"last_seen"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type SnapshotRequest struct {
	Serial string `json:"serial"`
}

type CommandRequest struct {
	Serial  string                 `json:"serial"`
	Command string                 `json:"command"` // ptz_left, ptz_right, ptz_up, ptz_down, ptz_zoom_in, ptz_zoom_out
	Params  map[string]interface{} `json:"params,omitempty"`
}
