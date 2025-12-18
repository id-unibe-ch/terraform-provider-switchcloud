// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectMemberResource{}
var _ resource.ResourceWithImportState = &ProjectMemberResource{}
var _ resource.ResourceWithValidateConfig = &ProjectMemberResource{}

func NewProjectMemberResource() resource.Resource {
	return &ProjectMemberResource{}
}

// ProjectResource defines the resource implementation.
type ProjectMemberResource struct {
	client   *http.Client
	endpoint string
}

// ProjectResourceModel describes the resource data model.
type ProjectMemberResourceModel struct {
	Id          types.String `tfsdk:"id"`
	ProjectId   types.String `tfsdk:"project_id"`
	UserId      types.String `tfsdk:"user_id"`
	EMail       types.String `tfsdk:"email"`
	DisplayName types.String `tfsdk:"display_name"`
}

// ProjectMember represents the API response structure.
type ProjectMember struct {
	Id        string            `json:"id"`
	ProjectId string            `json:"project_id"`
	UserId    string            `json:"user_id"`
	User      ProjectMemberUser `json:"user"`
}

type ProjectMemberUser struct {
	Id          string `json:"id"`
	EMail       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type ProjectMemberCreateRequest struct {
	UserId string `json:"user_id,omitempty"`
	EMail  string `json:"email,omitempty"`
}

func (r *ProjectMemberResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_member"
}

func (r *ProjectMemberResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A member of a project in the Switchcloud platform.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project member identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Project ID to which the member should be managed.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "User ID of the project member",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email of the project member",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Display name of the project member",
				Computed:            true,
			},
		},
	}
}

func (r *ProjectMemberResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ProjectMemberResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that either user_id or email is provided
	if config.UserId.IsNull() && config.EMail.IsNull() {
		resp.Diagnostics.AddError(
			"Configuration Error",
			"Either 'user_id' or 'email' must be provided for a project member.",
		)
	}

	if !config.UserId.IsNull() && !config.EMail.IsNull() {
		resp.Diagnostics.AddError(
			"Configuration Error",
			"Only one of 'user_id' or 'email' can be provided for a project member.",
		)
	}
}

func (r *ProjectMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected map[string]interface{}, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	client, ok := providerData["client"].(*http.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected HTTP Client Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", providerData["client"]),
		)
		return
	}

	endpoint, ok := providerData["endpoint"].(string)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Endpoint Type",
			fmt.Sprintf("Expected string, got: %T. Please report this issue to the provider developers.", providerData["endpoint"]),
		)
		return
	}

	r.client = client
	r.endpoint = endpoint
}

func (r *ProjectMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectMemberResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request body
	createRequest := ProjectMemberCreateRequest{}
	if data.UserId.IsUnknown() {
		createRequest.EMail = data.EMail.ValueString()
	} else {
		createRequest.UserId = data.UserId.ValueString()
	}

	// Marshal request body
	jsonBody, err := json.Marshal(createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal create request, got error: %s", err))
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", strings.TrimSuffix(r.endpoint, "/")+"/api/v1/projects/"+data.ProjectId.ValueString()+"/members", bytes.NewBuffer(jsonBody))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create request, got error: %s", err))
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Make API call
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project member, got error: %s", err))
		return
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read response body, got error: %s", err))
		return
	}

	// Check response status
	if httpResp.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)))
		return
	}

	// Parse response
	var projectMember ProjectMember
	if err := json.Unmarshal(body, &projectMember); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response, got error: %s", err))
		return
	}

	// Update model with response data
	data.Id = types.StringValue(projectMember.Id)
	data.ProjectId = types.StringValue(projectMember.ProjectId)
	data.UserId = types.StringValue(projectMember.UserId)
	data.EMail = types.StringValue(projectMember.User.EMail)
	data.DisplayName = types.StringValue(projectMember.User.DisplayName)

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a project member resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectMemberResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", strings.TrimSuffix(r.endpoint, "/")+"/api/v1/projects/"+data.ProjectId.ValueString()+"/members/"+data.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create request, got error: %s", err))
		return
	}

	httpReq.Header.Set("Accept", "application/json")

	// Make API call
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read response body, got error: %s", err))
		return
	}

	// Check if project member was deleted
	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	// Check response status
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)))
		return
	}

	// Parse response
	var projectMember ProjectMember
	if err := json.Unmarshal(body, &projectMember); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response, got error: %s", err))
		return
	}

	// Update model with response data
	data.Id = types.StringValue(projectMember.Id)
	data.ProjectId = types.StringValue(projectMember.ProjectId)
	data.UserId = types.StringValue(projectMember.UserId)
	data.EMail = types.StringValue(projectMember.User.EMail)
	data.DisplayName = types.StringValue(projectMember.User.DisplayName)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("API Error", "Unable to update project member, method is not implemented on server")
}

func (r *ProjectMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectMemberResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", strings.TrimSuffix(r.endpoint, "/")+"/api/v1/projects/"+data.ProjectId.ValueString()+"/members/"+data.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create request, got error: %s", err))
		return
	}

	httpReq.Header.Set("Accept", "application/json")

	// Make API call
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project member, got error: %s", err))
		return
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read response body, got error: %s", err))
		return
	}

	// Check response status
	if httpResp.StatusCode != http.StatusNoContent {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)))
		return
	}

	tflog.Trace(ctx, "deleted a project member resource")
}

func (r *ProjectMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ProjectMemberResourceModel

	// The import ID is expected to be in the format: project_id/member_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			"Expected import identifier with format: project_id/member_id. Got: "+req.ID,
		)
		return
	}

	projectId := parts[0]
	memberId := parts[1]
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), memberId)...)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", strings.TrimSuffix(r.endpoint, "/")+"/api/v1/projects/"+data.ProjectId.ValueString()+"/members/"+data.Id.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create request, got error: %s", err))
		return
	}

	httpReq.Header.Set("Accept", "application/json")

	// Make API call
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}
	defer httpResp.Body.Close()

	// Read response body
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read response body, got error: %s", err))
		return
	}

	// Check if project member was deleted
	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	// Check response status
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)))
		return
	}

	// Parse response
	var projectMember ProjectMember
	if err := json.Unmarshal(body, &projectMember); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response, got error: %s", err))
		return
	}

	// Update model with response data
	data.Id = types.StringValue(projectMember.Id)
	data.ProjectId = types.StringValue(projectMember.ProjectId)
	data.UserId = types.StringValue(projectMember.UserId)
	data.EMail = types.StringValue(projectMember.User.EMail)
	data.DisplayName = types.StringValue(projectMember.User.DisplayName)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
