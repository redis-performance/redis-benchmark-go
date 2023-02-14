package main

import (
	"context"
	radix "github.com/mediocregopher/radix/v4"
)

type Client interface {

	// Do performs an Action on a Conn connected to the redis instance.
	Do(context.Context, radix.Action) error

	// Once Close() is called all future method calls on the Client will return
	// an error
	Close() error
}
