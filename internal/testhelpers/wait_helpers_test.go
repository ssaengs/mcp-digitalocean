package testhelpers

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	dropletmocks "mcp-digitalocean/pkg/registry/droplet"
)

func TestWaitForAction_SucceedsWhenCompletes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockActions := dropletmocks.NewMockDropletActionsService(ctrl)

	dropletID := 123
	actionID := 1

	// First call returns in-progress, second returns completed
	gomock.InOrder(
		mockActions.EXPECT().
			Get(gomock.Any(), dropletID, actionID).
			Return(&godo.Action{ID: actionID, Status: "in-progress"}, nil, nil),
		mockActions.EXPECT().
			Get(gomock.Any(), dropletID, actionID).
			Return(&godo.Action{ID: actionID, Status: "completed"}, nil, nil),
	)

	client := &godo.Client{DropletActions: mockActions}

	ctx := context.Background()
	act, err := WaitForAction(ctx, client, dropletID, actionID, 5*time.Millisecond, 500*time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, act)
	require.Equal(t, "completed", act.Status)
}

func TestWaitForAction_ReturnsErrorWhenErrored(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockActions := dropletmocks.NewMockDropletActionsService(ctrl)

	dropletID := 123
	actionID := 2

	mockActions.EXPECT().
		Get(gomock.Any(), dropletID, actionID).
		Return(&godo.Action{ID: actionID, Status: "errored"}, nil, nil).
		Times(1)

	client := &godo.Client{DropletActions: mockActions}

	ctx := context.Background()
	act, err := WaitForAction(ctx, client, dropletID, actionID, 5*time.Millisecond, 500*time.Millisecond)
	require.Error(t, err)
	require.NotNil(t, act)
	require.Equal(t, "errored", act.Status)
}

func TestWaitForActions_WaitsForAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockActions := dropletmocks.NewMockDropletActionsService(ctrl)

	dropletID := 321
	actionIDs := []int{10, 11}

	// Both actions complete immediately
	mockActions.EXPECT().
		Get(gomock.Any(), dropletID, actionIDs[0]).
		Return(&godo.Action{ID: actionIDs[0], Status: "completed"}, nil, nil).
		Times(1)
	mockActions.EXPECT().
		Get(gomock.Any(), dropletID, actionIDs[1]).
		Return(&godo.Action{ID: actionIDs[1], Status: "completed"}, nil, nil).
		Times(1)

	client := &godo.Client{DropletActions: mockActions}

	ctx := context.Background()
	acts, err := WaitForActions(ctx, client, dropletID, actionIDs, 5*time.Millisecond, 500*time.Millisecond)
	require.NoError(t, err)
	require.Len(t, acts, 2)
	require.Equal(t, actionIDs[0], acts[0].ID)
	require.Equal(t, actionIDs[1], acts[1].ID)
}

func TestWaitForDroplet_WaitsUntilActivePredicate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDroplets := dropletmocks.NewMockDropletsService(ctrl)

	dropletID := 555

	// First call: new, no networks. Second call: active with IPv4.
	mockDroplets.EXPECT().
		Get(gomock.Any(), dropletID).
		Return(&godo.Droplet{ID: dropletID, Status: "new", Networks: &godo.Networks{V4: []godo.NetworkV4{}}}, nil, nil).
		Times(1)
	mockDroplets.EXPECT().
		Get(gomock.Any(), dropletID).
		Return(&godo.Droplet{ID: dropletID, Status: "active", Networks: &godo.Networks{V4: []godo.NetworkV4{{IPAddress: "1.2.3.4"}}}}, nil, nil).
		Times(1)

	client := &godo.Client{Droplets: mockDroplets}

	ctx := context.Background()
	d, err := WaitForDroplet(ctx, client, dropletID, IsDropletActive, 5*time.Millisecond, 500*time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, "active", d.Status)
	require.Len(t, d.Networks.V4, 1)
}

func TestWaitForDropletDeleted_Treats404AsSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDroplets := dropletmocks.NewMockDropletsService(ctrl)

	dropletID := 999

	// Simulate API returning not found via response with StatusCode 404.
	// The helpers check resp.StatusCode via the underlying http.Response.
	// Return nil droplet, a godo.Response that wraps an http.Response with 404, and an error to trigger the 404 branch.
	mockDroplets.EXPECT().
		Get(gomock.Any(), dropletID).
		Return(nil, &godo.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}, errors.New("not found")).
		Times(1)

	client := &godo.Client{Droplets: mockDroplets}

	ctx := context.Background()
	// WaitForDropletDeleted should return nil error when droplet not found.
	err := WaitForDropletDeleted(ctx, client, dropletID, 5*time.Millisecond, 500*time.Millisecond)
	require.NoError(t, err)
}
