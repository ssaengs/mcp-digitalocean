//go:build integration

package testing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const (
	defaultDIRegion                      = "tor1"
	defaultDIGPUSlug                     = "gpu-mi325x1-256gb"
	defaultDIModelSlug                   = "mistralai/Mistral-7B-Instruct-v0.3"
	defaultDITestTimeout                 = 2 * time.Hour
	diPollInterval                       = 30 * time.Second
	diIntermediateStateObservationWindow = 30 * time.Second
	diDeletePollTimeout                  = 10 * time.Minute
	diPostCreateScaleDelay               = 5 * time.Minute
	diUpdateRetryMax                     = 5
	diVPCListMaxPages                    = 5
	diVPCListPerPage                     = 50
)

type diCreateResponse struct {
	DedicatedInference *godo.DedicatedInference      `json:"dedicated_inference"`
	Token              *godo.DedicatedInferenceToken `json:"token,omitempty"`
}

// TestDedicatedInferenceList exercises the dedicated-inference MCP tools against the live API (list only; no create/delete).
func TestDedicatedInferenceList(t *testing.T) {
	t.Parallel()

	out := callTool[struct {
		Items []godo.DedicatedInferenceListItem `json:"items"`
		Meta  *godo.Meta                        `json:"meta,omitempty"`
	}](t, "dedicated-inference-list", map[string]any{})

	require.NotNil(t, out.Items)
	t.Logf("listed %d dedicated inference instance(s)", len(out.Items))
}

// TestDedicatedInferenceLifecycle exercises create -> active -> list -> scale -> delete via MCP tools.
func TestDedicatedInferenceLifecycle(t *testing.T) {
	ctx := context.Background()
	c := initializeClient(ctx, t)
	t.Cleanup(func() { c.Close() })

	vpcUUID := discoverVPCInRegion(t, c, defaultDIRegion)
	t.Logf("DI lifecycle config: region=%s gpu=%s model=%s vpc=%s",
		defaultDIRegion, defaultDIGPUSlug, defaultDIModelSlug, vpcUUID)

	name := diCanaryResourceName(defaultDIRegion)
	createArgs := map[string]any{
		"Name":                 name,
		"Region":               defaultDIRegion,
		"VPCUUID":              vpcUUID,
		"EnablePublicEndpoint": true,
		"ModelDeployments":     diCreateModelDeployments(defaultDIModelSlug, defaultDIGPUSlug, 1),
	}

	var diID string

	t.Run("create", func(st *testing.T) {
		raw, resp := callDIToolRaw(st, c, "dedicated-inference-create", createArgs)
		require.False(st, resp.IsError, "create failed: %s", raw)

		var created diCreateResponse
		require.NoError(st, json.Unmarshal([]byte(raw), &created))
		require.NotNil(st, created.DedicatedInference, "create response missing dedicated_inference")
		require.NotEmpty(st, created.DedicatedInference.ID, "create response missing id")

		diID = created.DedicatedInference.ID
		cleanupDI(t, c, diID)

		require.Equal(st, name, created.DedicatedInference.Name)
		require.Equal(st, defaultDIRegion, created.DedicatedInference.Region)
		require.True(st, diStatusAllowed(created.DedicatedInference.Status),
			"unexpected status after create: %s", created.DedicatedInference.Status)
		require.NotNil(st, created.Token, "create response missing token")
		require.NotEmpty(st, created.Token.ID)
		require.NotEmpty(st, created.Token.Value)
	})

	require.NotEmpty(t, diID, "create subtest must set diID before lifecycle continues")
	t.Logf("created dedicated inference id=%s name=%s region=%s — look for this in the control panel or API while the test runs",
		diID, name, defaultDIRegion)

	t.Run("wait_for_active", func(t *testing.T) {
		active := pollDIActive(t, c, diID, name, "provisioning", defaultDITestTimeout)
		require.NotNil(t, active.Endpoints)
		require.NotEmpty(t, active.Endpoints.PublicEndpointFQDN)
		require.NotNil(t, active.DeploymentSpec)
		require.NotEmpty(t, active.DeploymentSpec.ModelDeployments)
		require.Equal(t, uint64(1), active.DeploymentSpec.ModelDeployments[0].Accelerators[0].Scale)
		require.Equal(t, vpcUUID, active.VPCUUID, "VPCUUID round-trip mismatch")

		t.Run("list_filtered_and_paginated", func(t *testing.T) {
			filtered := listDIViaMCP(t, c, map[string]any{"Name": name})
			require.Len(t, filtered, 1)
			require.Equal(t, diID, filtered[0].ID)

			page1, meta := listDIWithMeta(t, c, map[string]any{
				"Page":    float64(1),
				"PerPage": float64(1),
			})
			require.Len(t, page1, 1, "Page=1 PerPage=1 should return one item")
			if meta != nil {
				require.GreaterOrEqual(t, meta.Total, 1)
			}
		})

		t.Run("update_scale", func(t *testing.T) {
			modelIDBefore := active.DeploymentSpec.ModelDeployments[0].ModelID
			regionBefore := active.Region
			vpcBefore := active.VPCUUID

			t.Logf("waiting %s after activation before scale (regional propagation)", diPostCreateScaleDelay)
			time.Sleep(diPostCreateScaleDelay)

			_ = updateDIScaleWithRetry(t, c, active, 2)
			scaled := pollDIActive(t, c, diID, name, "updating", defaultDITestTimeout)
			require.Equal(t, uint64(2), scaled.DeploymentSpec.ModelDeployments[0].Accelerators[0].Scale)
			require.Equal(t, modelIDBefore, scaled.DeploymentSpec.ModelDeployments[0].ModelID)
			require.Equal(t, regionBefore, scaled.Region)
			require.Equal(t, vpcBefore, scaled.VPCUUID)
			require.Equal(t, "active", strings.ToLower(scaled.Status))
		})

		t.Run("delete", func(t *testing.T) {
			raw, resp := callDIToolRaw(t, c, "dedicated-inference-delete", map[string]any{
				"DedicatedInferenceID": diID,
			})
			require.False(t, resp.IsError, "delete failed: %s", raw)

			var deleteResp struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal([]byte(raw), &deleteResp), "delete response not JSON: %s", raw)
			require.Equal(t, "success", deleteResp.Status, "unexpected delete status; full body: %s", raw)

			pollDIDeleted(t, c, diID)
		})
	})
}

func discoverVPCInRegion(t *testing.T, c *client.Client, region string) string {
	t.Helper()

	var fallback string
	for page := 1; page <= diVPCListMaxPages; page++ {
		raw, resp := callDIToolRaw(t, c, "vpc-list", map[string]any{
			"Page":    float64(page),
			"PerPage": float64(diVPCListPerPage),
		})
		require.False(t, resp.IsError, "vpc-list failed: %s", raw)

		var vpcs []godo.VPC
		require.NoError(t, json.Unmarshal([]byte(raw), &vpcs))
		for _, vpc := range vpcs {
			if vpc.RegionSlug != region {
				continue
			}
			if vpc.Default {
				return vpc.ID
			}
			if fallback == "" {
				fallback = vpc.ID
			}
		}
		if len(vpcs) < diVPCListPerPage {
			break
		}
	}
	require.NotEmpty(t, fallback, "no VPC found in region %s via vpc-list", region)
	return fallback
}

func diCanaryResourceName(region string) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("mcp-canary-%s-%d-%s", region, time.Now().Unix(), hex.EncodeToString(b))
}

func diCreateModelDeployments(modelSlug, gpuSlug string, scale uint64) []map[string]any {
	return []map[string]any{
		{
			"ModelSlug":     modelSlug,
			"ModelProvider": "hugging_face",
			"Accelerators": []map[string]any{
				{
					"AcceleratorSlug": gpuSlug,
					"Scale":           float64(scale),
					"Type":            "prefill_decode",
				},
			},
		},
	}
}

func diModelDeploymentsFromDI(t *testing.T, di *godo.DedicatedInference, scale uint64) []map[string]any {
	t.Helper()
	require.NotNil(t, di.DeploymentSpec, "deployment spec")
	require.NotEmpty(t, di.DeploymentSpec.ModelDeployments, "model deployments")

	var out []map[string]any
	for _, md := range di.DeploymentSpec.ModelDeployments {
		entry := map[string]any{
			"ModelSlug":     md.ModelSlug,
			"ModelProvider": md.ModelProvider,
		}
		if md.ModelID != "" {
			entry["ModelID"] = md.ModelID
		}
		var accs []map[string]any
		for i, acc := range md.Accelerators {
			s := scale
			if i > 0 {
				s = acc.Scale
			}
			accs = append(accs, map[string]any{
				"AcceleratorSlug": acc.AcceleratorSlug,
				"Scale":           float64(s),
				"Type":            acc.Type,
			})
		}
		entry["Accelerators"] = accs
		out = append(out, entry)
	}
	return out
}

func callDIToolRaw(t *testing.T, c *client.Client, name string, args map[string]any) (string, *mcp.CallToolResult) {
	t.Helper()
	start := time.Now()
	ctx := context.Background()

	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: name, Arguments: args},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	var raw string
	if len(resp.Content) > 0 {
		if tc, ok := resp.Content[0].(mcp.TextContent); ok {
			raw = tc.Text
		}
	}

	t.Logf("[DI tool] %s args=%v latency=%s is_error=%v", name, args, time.Since(start), resp.IsError)
	if resp.IsError {
		t.Logf("[DI tool] %s error body: %s", name, raw)
	}
	return raw, resp
}

func pollDIActive(t *testing.T, c *client.Client, diID, diName, intermediateStatus string, timeout time.Duration) *godo.DedicatedInference {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var (
		activeDI             *godo.DedicatedInference
		seenIntermediateGet  bool
		seenIntermediateList bool
		firstActiveAt        time.Time
	)

	for time.Now().Before(deadline) {
		di := getDIViaMCP(t, c, diID)
		status := strings.ToLower(di.Status)

		if status == intermediateStatus {
			seenIntermediateGet = true
		}

		if !seenIntermediateList {
			for _, item := range listDIViaMCP(t, c, map[string]any{"Name": diName}) {
				if item.ID == diID && strings.ToLower(item.Status) == intermediateStatus {
					seenIntermediateList = true
					break
				}
			}
		}

		if status != "active" {
			t.Logf("dedicated inference %s status=%s, waiting for active ...", diID, di.Status)
			time.Sleep(diPollInterval)
			continue
		}

		if firstActiveAt.IsZero() {
			firstActiveAt = time.Now()
		}

		if seenIntermediateGet && seenIntermediateList {
			activeDI = di
			break
		}

		if time.Since(firstActiveAt) < diIntermediateStateObservationWindow {
			t.Logf("dedicated inference %s active but %s not fully observed (get=%v list=%v), within observation window ...",
				diID, intermediateStatus, seenIntermediateGet, seenIntermediateList)
			time.Sleep(diPollInterval)
			continue
		}

		t.Logf("dedicated inference %s active; %s not observed within %s (get=%v list=%v) - accepting",
			diID, intermediateStatus, diIntermediateStateObservationWindow, seenIntermediateGet, seenIntermediateList)
		activeDI = di
		break
	}

	require.NotNil(t, activeDI, "dedicated inference %s did not become active within %s", diID, timeout)
	return activeDI
}

func getDIViaMCP(t *testing.T, c *client.Client, diID string) *godo.DedicatedInference {
	t.Helper()
	raw, resp := callDIToolRaw(t, c, "dedicated-inference-get", map[string]any{
		"DedicatedInferenceID": diID,
	})
	require.False(t, resp.IsError, "get dedicated inference failed: %s", raw)

	var di godo.DedicatedInference
	require.NoError(t, json.Unmarshal([]byte(raw), &di))
	return &di
}

func listDIViaMCP(t *testing.T, c *client.Client, opts map[string]any) []godo.DedicatedInferenceListItem {
	t.Helper()
	items, _ := listDIWithMeta(t, c, opts)
	return items
}

func listDIWithMeta(t *testing.T, c *client.Client, opts map[string]any) ([]godo.DedicatedInferenceListItem, *godo.Meta) {
	t.Helper()
	if opts == nil {
		opts = map[string]any{}
	}
	raw, resp := callDIToolRaw(t, c, "dedicated-inference-list", opts)
	require.False(t, resp.IsError, "list dedicated inferences failed: %s", raw)

	var out struct {
		Items []godo.DedicatedInferenceListItem `json:"items"`
		Meta  *godo.Meta                        `json:"meta,omitempty"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &out))
	return out.Items, out.Meta
}

func pollDIDeleted(t *testing.T, c *client.Client, diID string) {
	t.Helper()
	deadline := time.Now().Add(diDeletePollTimeout)

	for time.Now().Before(deadline) {
		_, resp := callDIToolRaw(t, c, "dedicated-inference-get", map[string]any{
			"DedicatedInferenceID": diID,
		})
		if resp.IsError {
			var errText string
			if len(resp.Content) > 0 {
				if tc, ok := resp.Content[0].(mcp.TextContent); ok {
					errText = tc.Text
				}
			}
			lower := strings.ToLower(errText)
			if strings.Contains(errText, "404") || strings.Contains(lower, "not found") {
				t.Logf("dedicated inference %s deleted (404)", diID)
				return
			}
			t.Logf("get after delete returned error (retrying): %s", errText)
			time.Sleep(diPollInterval)
			continue
		}
		t.Logf("dedicated inference %s still present after delete, polling ...", diID)
		time.Sleep(diPollInterval)
	}
	t.Fatalf("dedicated inference %s still present after %s", diID, diDeletePollTimeout)
}

func updateDIScaleWithRetry(t *testing.T, c *client.Client, di *godo.DedicatedInference, scale uint64) *godo.DedicatedInference {
	t.Helper()
	require.NotNil(t, di.DeploymentSpec)

	args := map[string]any{
		"DedicatedInferenceID": di.ID,
		"Name":                 di.Name,
		"Region":               di.Region,
		"EnablePublicEndpoint": di.DeploymentSpec.EnablePublicEndpoint,
		"VPCUUID":              di.VPCUUID,
		"ModelDeployments":     diModelDeploymentsFromDI(t, di, scale),
	}

	var lastErr string
	for attempt := 0; attempt < diUpdateRetryMax; attempt++ {
		raw, resp := callDIToolRaw(t, c, "dedicated-inference-update", args)
		if !resp.IsError {
			var updated godo.DedicatedInference
			require.NoError(t, json.Unmarshal([]byte(raw), &updated))
			return &updated
		}
		lastErr = raw
		if strings.Contains(strings.ToLower(raw), "existing operation in progress") {
			t.Logf("update scale attempt %d: operation in progress, retrying after %s ...", attempt+1, diPostCreateScaleDelay)
			time.Sleep(diPostCreateScaleDelay)
			continue
		}
		require.Failf(t, "dedicated-inference-update failed", "%s", raw)
	}
	require.Failf(t, "dedicated-inference-update exhausted retries", "after %d retries: %s", diUpdateRetryMax, lastErr)
	return nil
}

func cleanupDI(t *testing.T, c *client.Client, diID string) {
	t.Helper()
	t.Cleanup(func() {
		t.Logf("cleanup: deleting dedicated inference %s", diID)
		_, resp := callDIToolRaw(t, c, "dedicated-inference-delete", map[string]any{
			"DedicatedInferenceID": diID,
		})
		if resp != nil && !resp.IsError {
			t.Logf("cleanup: MCP delete succeeded for %s", diID)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		token := os.Getenv("DIGITALOCEAN_API_TOKEN")
		if token == "" {
			t.Logf("cleanup: MCP delete failed and DIGITALOCEAN_API_TOKEN unset, cannot godo fallback")
			return
		}

		oauthClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
		gclient, err := godo.New(oauthClient, godo.SetUserAgent("mcp-e2e-di-cleanup"))
		if err != nil {
			t.Logf("cleanup: godo client error: %v", err)
			return
		}

		_, err = gclient.DedicatedInference.Delete(ctx, diID)
		if err != nil {
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") {
				t.Logf("cleanup: godo delete %s already gone", diID)
				return
			}
			t.Logf("cleanup: godo delete failed for %s: %v", diID, err)
			return
		}
		t.Logf("cleanup: godo delete succeeded for %s", diID)
	})
}

func diStatusAllowed(status string) bool {
	switch strings.ToLower(status) {
	case "new", "provisioning", "active":
		return true
	default:
		return false
	}
}
