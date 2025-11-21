package testhelpers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/digitalocean/godo"
)

// Default timeouts
const (
	defaultInterval = 2 * time.Second
	defaultTimeout  = 10 * time.Minute
)

// WaitForAction polls for a droplet action to complete or error.
func WaitForAction(ctx context.Context, client *godo.Client, dropletID, actionID int, interval, timeout time.Duration) (*godo.Action, error) {
	var action *godo.Action
	err := poll(ctx, interval, timeout, func() (bool, error) {
		a, resp, err := client.DropletActions.Get(ctx, dropletID, actionID)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, errors.New("action not found")
		}
		if err != nil {
			return false, err
		}
		action = a
		switch a.Status {
		case "completed":
			return true, nil
		case "errored":
			return false, errors.New("action errored")
		default:
			return false, nil // in-progress
		}
	})
	return action, err
}

// WaitForActions waits for multiple actions sequentially.
func WaitForActions(ctx context.Context, client *godo.Client, dropletID int, actionIDs []int, interval, timeout time.Duration) ([]*godo.Action, error) {
	results := make([]*godo.Action, 0, len(actionIDs))
	for _, aid := range actionIDs {
		act, err := WaitForAction(ctx, client, dropletID, aid, interval, timeout)
		if err != nil {
			return nil, err
		}
		results = append(results, act)
	}
	return results, nil
}

// WaitForDroplet polls until the predicate returns true.
// If predicate is nil, it returns (nil, nil) immediately upon a 404 (used for deletion checks).
func WaitForDroplet(ctx context.Context, client *godo.Client, dropletID int, predicate func(*godo.Droplet) bool, interval, timeout time.Duration) (*godo.Droplet, error) {
	var last *godo.Droplet
	err := poll(ctx, interval, timeout, func() (bool, error) {
		d, resp, err := client.Droplets.Get(ctx, dropletID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				if predicate == nil {
					last = nil
					return true, nil // Deletion confirmed
				}
				return false, errors.New("droplet not found")
			}
			return false, err
		}
		last = d
		if predicate != nil && predicate(d) {
			return true, nil
		}
		return false, nil
	})
	return last, err
}

// WaitForDropletDeleted checks for 404 status.
func WaitForDropletDeleted(ctx context.Context, client *godo.Client, dropletID int, interval, timeout time.Duration) error {
	_, err := WaitForDroplet(ctx, client, dropletID, nil, interval, timeout)
	return err
}

// IsDropletActive checks if status is active and has an IPv4.
func IsDropletActive(d *godo.Droplet) bool {
	return d != nil && d.Status == "active" && d.Networks != nil && len(d.Networks.V4) > 0
}

// MustGodoClient returns a client or panics if the token is missing.
func MustGodoClient() *godo.Client {
	token := os.Getenv("DIGITALOCEAN_API_TOKEN")
	if token == "" {
		panic("DIGITALOCEAN_API_TOKEN environment variable must be set to run E2E tests")
	}
	return godo.NewFromToken(token)
}

// --- Internal Helpers ---

// poll is a generic loop that runs 'check' every 'interval' until 'timeout'.
// check() returns (done, error). If done=true, poll returns nil. If error!=nil, poll returns error.
func poll(ctx context.Context, interval, timeout time.Duration, check func() (bool, error)) error {
	if interval == 0 {
		interval = defaultInterval
	}
	if timeout == 0 {
		timeout = defaultTimeout
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("timed out after %s", timeout)
		case <-ticker.C:
			// fallthrough to check
		default:
			// Check immediately on first run
		}

		done, err := check()
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		// Ensure we wait for the ticker if we just ran a check to avoid hot looping
		if interval > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return fmt.Errorf("timed out after %s", timeout)
			case <-ticker.C:
				continue
			}
		}
	}
}
