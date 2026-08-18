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
var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// ProjectResource defines the resource implementation.
type ProjectResource struct {
	client   *http.Client
	endpoint string
}

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	BillingReference types.String `tfsdk:"billing_reference"`
	TenantId         types.String `tfsdk:"tenant_id"`
	Archived         types.Bool   `tfsdk:"archived"`
	ArchivedAt       types.String `tfsdk:"archived_at"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

// Project represents the API response structure.
type Project struct {
	Id                       string  `json:"id"`
	Name                     string  `json:"name"`
	Description              *string `json:"description,omitempty"`
	CustomerBillingReference *string `json:"customer_billing_reference,omitempty"`
	TenantId                 string  `json:"tenant_id"`
	Archived                 bool    `json:"archived"`
	ArchivedAt               string  `json:"archived_at,omitempty"`
	CreatedAt                string  `json:"created_at"`
	UpdatedAt                string  `json:"updated_at"`
}

// ProjectRequestFields represents the mutable project fields accepted by the
// create and update endpoints.
type ProjectRequestFields struct {
	Name                     string  `json:"name,omitempty"`
	Description              *string `json:"description,omitempty"`
	CustomerBillingReference *string `json:"customer_billing_reference,omitempty"`
}

// ProjectRequestBody is the wrapped request body expected by the
// create/update project endpoints, e.g. {"project": {...}}.
type ProjectRequestBody struct {
	Project ProjectRequestFields `json:"project"`
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "A project in the Switchcloud platform.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Project name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Project description",
				Optional:            true,
			},
			"billing_reference": schema.StringAttribute{
				MarkdownDescription: "Customer billing reference for the project",
				Optional:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "Tenant ID that owns this project",
				Computed:            true,
			},
			"archived": schema.BoolAttribute{
				MarkdownDescription: "Whether the project is archived",
				Computed:            true,
			},
			"archived_at": schema.StringAttribute{
				MarkdownDescription: "When the project was archived",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "When the project was created",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "When the project was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// projectRequestFieldsFromModel builds the mutable project fields sent to
// the create/update endpoints from the Terraform model.
func projectRequestFieldsFromModel(data ProjectResourceModel) ProjectRequestFields {
	fields := ProjectRequestFields{
		Name: data.Name.ValueString(),
	}

	if !data.Description.IsNull() {
		description := data.Description.ValueString()
		fields.Description = &description
	}

	if !data.BillingReference.IsNull() {
		billingReference := data.BillingReference.ValueString()
		fields.CustomerBillingReference = &billingReference
	}

	return fields
}

// applyProjectToModel copies the API response fields into the Terraform model.
func applyProjectToModel(data *ProjectResourceModel, project Project) {
	data.Id = types.StringValue(project.Id)
	data.Name = types.StringValue(project.Name)
	data.Description = types.StringPointerValue(project.Description)
	data.BillingReference = types.StringPointerValue(project.CustomerBillingReference)
	data.TenantId = types.StringValue(project.TenantId)
	data.Archived = types.BoolValue(project.Archived)
	data.ArchivedAt = types.StringValue(project.ArchivedAt)
	data.CreatedAt = types.StringValue(project.CreatedAt)
	data.UpdatedAt = types.StringValue(project.UpdatedAt)
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create API request body
	createRequest := ProjectRequestBody{Project: projectRequestFieldsFromModel(data)}

	// Marshal request body
	jsonBody, err := json.Marshal(createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal create request, got error: %s", err))
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", strings.TrimSuffix(r.endpoint, "/")+"/api/v1/projects", bytes.NewBuffer(jsonBody))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create request, got error: %s", err))
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Make API call
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
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
	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response, got error: %s", err))
		return
	}

	// Update model with response data
	applyProjectToModel(&data, project)

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a project resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", strings.TrimSuffix(r.endpoint, "/")+"/api/v1/projects/"+data.Id.ValueString(), nil)
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

	// Check if project was deleted
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
	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response, got error: %s", err))
		return
	}

	// Update model with response data
	applyProjectToModel(&data, project)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request body
	updateRequest := ProjectRequestBody{Project: projectRequestFieldsFromModel(data)}

	// Marshal request body
	jsonBody, err := json.Marshal(updateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal update request, got error: %s", err))
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", strings.TrimSuffix(r.endpoint, "/")+"/api/v1/projects/"+data.Id.ValueString(), bytes.NewBuffer(jsonBody))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create update request, got error: %s", err))
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Make API call
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
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
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)))
		return
	}

	// Parse response
	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response, got error: %s", err))
		return
	}

	// Update model with response data
	applyProjectToModel(&data, project)

	// Write logs using the tflog package
	tflog.Trace(ctx, "updated a project resource")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", strings.TrimSuffix(r.endpoint, "/")+"/api/v1/projects/"+data.Id.ValueString()+"/archive", nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create archive request, got error: %s", err))
		return
	}

	httpReq.Header.Set("Accept", "application/json")

	// Make API call
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to archive project, got error: %s", err))
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
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("API returned status %d: %s", httpResp.StatusCode, string(body)))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
