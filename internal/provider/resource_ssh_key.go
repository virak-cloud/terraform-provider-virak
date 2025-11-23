package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/virak-cloud/cli/pkg/http"
	"github.com/virak-cloud/terraform-provider-virak/internal/models"
)

var _ resource.Resource = &sshKeyResource{}

func NewSSHKeyResource(mutex *sync.Mutex) resource.Resource {
	return &sshKeyResource{mutex: mutex}
}

type sshKeyResource struct {
	client *http.Client
	mutex  *sync.Mutex
}

func (r *sshKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *sshKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Virak Cloud SSH key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the SSH key, retrieved from the SSH key list after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the SSH key.",
			},
			"public_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           false,
				MarkdownDescription: "The public SSH key content.",
			},
		},
	}
}

func (r *sshKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *sshKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.SSHKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: Get list of existing SSH keys and save it
	initialKeysResp, err := r.client.ListUserSSHKeys()
	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Key Listing Failed",
			fmt.Sprintf("Failed to list SSH keys before creation. Error: %s", err),
		)
		return
	}

	// Step 2: Check if SSH key with same name already exists - return error if found
	desiredName := data.Name.ValueString()
	for _, key := range initialKeysResp.UserData {
		if key.DisplayName == desiredName {
			resp.Diagnostics.AddError(
				"SSH Key Already Exists",
				fmt.Sprintf("SSH key with name '%s' already exists (ID: %s). Please choose a unique name.", desiredName, key.ID),
			)
			return
		}
	}

	// Step 3: Name doesn't exist, proceed with creation
	// Create response only returns Success: true, so we'll find the key by name in the polling loop
	_, err = r.client.AddUserSSHKey(data.Name.ValueString(), data.PublicKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Key Creation Failed",
			fmt.Sprintf("Failed to create SSH key '%s'. Error: %s", data.Name.ValueString(), err),
		)
		return
	}

	// Step 4: Poll for the created key with timeout of 600 seconds, checking every 5 seconds
	// Since create response only returns Success: true, we compare by name to find the created key
	timeout := 600 * time.Second
	checkInterval := 5 * time.Second
	startTime := time.Now()

	for {
		// Check if timeout has been reached
		if time.Since(startTime) >= timeout {
			resp.Diagnostics.AddError(
				"SSH Key Creation Timeout",
				fmt.Sprintf("SSH key '%s' was created but not found in the list within %d seconds.", desiredName, int(timeout.Seconds())),
			)
			return
		}

		// Get current list of SSH keys
		currentKeysResp, err := r.client.ListUserSSHKeys()
		if err != nil {
			// If listing fails, wait and retry
			time.Sleep(checkInterval)
			continue
		}

		// Check if the created key exists in the list by comparing name
		var foundKey *models.SSHKeyResourceModel
		for _, key := range currentKeysResp.UserData {
			if key.DisplayName == desiredName {
				foundKey = &models.SSHKeyResourceModel{
					ID:        types.StringValue(key.ID),
					Name:      types.StringValue(key.DisplayName),
					PublicKey: types.StringValue(key.DataValue),
				}
				break
			}
		}

		// If key found, set state and return
		if foundKey != nil {
			data = *foundKey
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}

		// Key not found yet, wait before next check
		time.Sleep(checkInterval)
	}
}

func (r *sshKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.SSHKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keysResp, err := r.client.ListUserSSHKeys()
	if err != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	var foundKey *models.SSHKeyResourceModel
	for _, key := range keysResp.UserData {
		if key.ID == data.ID.ValueString() {
			foundKey = &models.SSHKeyResourceModel{
				ID:        types.StringValue(key.ID),
				Name:      types.StringValue(key.DisplayName),
				PublicKey: types.StringValue(key.DataValue),
			}
			break
		}
	}

	if foundKey == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data = *foundKey
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *sshKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"SSH key resources do not support updates. To change an SSH key, delete and recreate it.",
	)
}

func (r *sshKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var data models.SSHKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: List existing SSH keys to check if key still exists
	keysResp, err := r.client.ListUserSSHKeys()
	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Key Listing Failed",
			fmt.Sprintf("Failed to list SSH keys before deletion. Error: %s", err),
		)
		return
	}

	// Step 2: Check if SSH key exists by ID (from state)
	keyID := data.ID.ValueString()
	keyFound := false
	for _, key := range keysResp.UserData {
		if key.ID == keyID {
			keyFound = true
			break
		}
	}

	// Step 3: If key not found, consider deletion successful (idempotent)
	if !keyFound {
		// Key doesn't exist, deletion already completed or never existed
		return
	}

	// Step 4: Key exists, proceed with deletion
	_, err = r.client.DeleteUserSSHKey(keyID)
	if err != nil {
		resp.Diagnostics.AddError(
			"SSH Key Deletion Failed",
			fmt.Sprintf("Failed to delete SSH key '%s'. Error: %s", keyID, err),
		)
		return
	}
}
