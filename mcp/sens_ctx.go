package mcp

import (
	"context"
	"sync"
)

type sensBag struct {
	mu  sync.Mutex
	ids []string
}

type sensCollectKey struct{}

func withSensCollect(ctx context.Context) (context.Context, *sensBag) {
	if ctx == nil {
		ctx = context.Background()
	}
	if existing, ok := ctx.Value(sensCollectKey{}).(*sensBag); ok && existing != nil {
		return ctx, existing
	}
	bag := &sensBag{}
	return context.WithValue(ctx, sensCollectKey{}, bag), bag
}

func noteSensID(ctx context.Context, id string) {
	if ctx == nil || id == "" {
		return
	}
	bag, ok := ctx.Value(sensCollectKey{}).(*sensBag)
	if !ok || bag == nil {
		return
	}
	bag.mu.Lock()
	defer bag.mu.Unlock()
	for _, x := range bag.ids {
		if x == id {
			return
		}
	}
	bag.ids = append(bag.ids, id)
}

func sensIDsFromCtx(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	bag, ok := ctx.Value(sensCollectKey{}).(*sensBag)
	if !ok || bag == nil {
		return nil
	}
	bag.mu.Lock()
	defer bag.mu.Unlock()
	out := make([]string, len(bag.ids))
	copy(out, bag.ids)
	return out
}
