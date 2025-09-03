package store

import "time"

type ClickEvent struct {
	Code string
	IP   string
	UA   string
	Ts   time.Time
}

type Stats struct {
	Code        string `json:"code"`
	Target      string `json:"target"`
	TotalClicks int64  `json:"totalClicks"`
	UniqueIPs   int64  `json:"uniqueIPs"`
	LastAccess  string `json:"lastAccess,omitempty"`
}

type Store interface {
	Save(code, target string) error
	Get(code string) (string, error)
	InsertClick(ev ClickEvent) error
	Stats(code string) (Stats, error)
	TopCodes(n int) ([]string, error)
}
