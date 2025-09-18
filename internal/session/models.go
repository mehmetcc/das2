package session

import (
	"time"

	"github.com/mehmetcc/das2/internal/httpx"
)

type SessionSummary struct {
	ID         string         `json:"id"`
	PersonID   int64          `json:"-"`
	DeviceID   string         `json:"device_id"`
	DeviceName string         `json:"device_name,omitempty"`
	Platform   httpx.Platform `json:"platform,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	LastUsedIP string         `json:"last_used_ip,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
}
