//go:build !darwin

package xbar

import (
	"context"
	"fmt"
)

func Render(ctx context.Context, opts *Options) error {
	return fmt.Errorf("xbar not supported on current platform")
}
