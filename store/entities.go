package store

import "errors"

var ErrEntityAlreadyExists = errors.New("entity already exists")

type Track struct {
	ID   int64
	Name string
}

type GlobalCounters struct {
	Tracks int64
}
