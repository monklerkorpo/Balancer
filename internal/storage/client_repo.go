package storage

import "errors"

// ClientLimit — модель для CRUD:
type ClientLimit struct {
    ClientID   string `json:"client_id"`
    Capacity   int    `json:"capacity"`
    RefillRate int    `json:"rate_per_sec"`
}

var ErrNotFound = errors.New("client not found")

// ClientRepository — CRUD для ClientLimit
type ClientRepository interface {
    Create(ClientLimit) error
    Get(id string) (ClientLimit, error)
    Update(ClientLimit) error
    Delete(id string) error
    List() ([]ClientLimit, error)
}
