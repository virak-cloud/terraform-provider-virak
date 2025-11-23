package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/virak-cloud/cli/pkg/http"
)

// baseResource provides common functionality for all resources
type baseResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

// Configure sets up the resource with the client from the provider
func (r *baseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*http.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

// setMutex sets the mutex for the resource
func (r *baseResource) setMutex(mutex *sync.Mutex) {
	r.mutex = mutex
}
