package db

import "time"

type Request struct {
	Id        int
	Path      string
	CreatedAt time.Time
}
