package ghapp

import (
	"context"
	"fmt"

	"github.com/isometry/ghait"
)

type contextKey struct{}

type notFoundError struct{}

func (notFoundError) Error() string {
	return "no GHApp was present"
}

func (notFoundError) IsNotFound() bool {
	return true
}

func NewContext(ctx context.Context, ghapp ghait.GHAIT) context.Context {
	return context.WithValue(ctx, contextKey{}, ghapp)
}

func FromContext(ctx context.Context) (ghait.GHAIT, error) {
	v := ctx.Value(contextKey{})
	if v == nil {
		return nil, notFoundError{}
	}

	switch v := v.(type) {
	case ghait.GHAIT:
		return v, nil
	default:
		return nil, fmt.Errorf("unexpected value type for ghapp context key: %T", v)
	}
}
