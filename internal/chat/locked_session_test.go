package chat

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestLockedSession_SerialisesParallelMutations is a regression test for the
// lost-write race between parallel add_to_draft calls within a single ReAct
// iteration. Without lockedSession both goroutines load an empty Draft,
// append their slot, and Set — last write wins, the other slot is lost.
//
// We simulate two mutator handlers running in parallel against a store that
// returns fresh *Session copies on every Get (the DBSessionStore property).
// With LockFor + lockedSession, both appends survive.
func TestLockedSession_SerialisesParallelMutations(t *testing.T) {
	store := newFreshLoadingStore()
	const sid = "race-1"

	// Seed an empty session so Get works.
	_ = store.GetOrCreate(sid)
	ctx := context.WithValue(context.Background(), sessionIDKey{}, sid)

	mutate := func(name string) {
		sess, unlock := lockedSession(ctx, store)
		defer unlock()
		// Mimic some realistic intra-handler latency to widen the race
		// window: any concurrent goroutine that read Draft first would
		// overwrite us with its own snapshot if no lock were held.
		time.Sleep(2 * time.Millisecond)
		sess.Draft = append(sess.Draft, DraftEntry{
			CustomerName:   name,
			ProductName:    "x",
			IdempotencyKey: name,
		})
		store.Set(sess)
	}

	var wg sync.WaitGroup
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			mutate(name)
		}(n)
	}
	wg.Wait()

	final := store.GetOrCreate(sid)
	if got := len(final.Draft); got != 8 {
		t.Fatalf("len(Draft) = %d, want 8 (lost write race)", got)
	}
}
