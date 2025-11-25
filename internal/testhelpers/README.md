# Test Helpers: Polling & Wait Logic

Provides standard, type-safe polling primitives for integration tests to replace boilerplate loops. Handles timeouts, context cancellation, and API retries consistently.

## Key Features

  * **Generic Polling:** Type-safe waiting for any resource (Droplets, Databases, etc.) using `func(*T) bool` predicates.
  * **Action Waiters:** Specialized handling for async operations to reach `completed` or `errored` states.
  * **Deletion Checks:** Treats `404 Not Found` responses as success.
  * **Smart Defaults:** Passing `0` for interval or timeout uses sensible, CI-safe values.

-----

## Usage

### Resource State

Polls until the predicate returns true.

```go
// Wait for a Droplet to become active
droplet, err := testhelpers.WaitForDroplet(
    ctx, client, id,
    testhelpers.IsDropletActive, // Predicate
    0, 0, // Use defaults
)
```

### Async Actions

Polls until the Action status is `completed`.

```go
action, err := testhelpers.WaitForAction(ctx, client, resourceID, actionID, 0, 0)
```

### Deletion Verification

Polls until the resource returns `404 Not Found`.

```go
if err := testhelpers.WaitForImageDeleted(ctx, client, imageID, 0, 0); err != nil {
    t.Errorf("Resource not deleted: %v", err)
}
```

-----

## Extending

To support new resources, wrap the internal `waitForResource` generic. Provide an API fetch closure and a predicate.

**Example: Adding Database Support**

```go
func WaitForDatabase(ctx context.Context, client *godo.Client, dbID string, pred func(*godo.Database) bool) (*godo.Database, error) {
    return waitForResource(ctx, 0, 0,
        // API Fetch
        func() (*godo.Database, *godo.Response, error) {
            return client.Databases.Get(ctx, dbID)
        },
        // Predicate
        pred,
    )
}
```

**Example: Adding Action Support**

```go
func WaitForReservedIPAction(ctx context.Context, client *godo.Client, ip string, actionID int) (*godo.Action, error) {
    return waitForActionGeneric(ctx, 0, 0, func() (*godo.Action, *godo.Response, error) {
        return client.ReservedIPActions.Get(ctx, ip, actionID)
    })
}
```

-----
