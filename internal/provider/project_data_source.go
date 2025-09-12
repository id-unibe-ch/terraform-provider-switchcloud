// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ProjectDataSource{}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

// ProjectDataSource defines the data source implementation.
type ProjectDataSource struct {
	client   *http.Client
	endpoint string
}

// ProjectDataSourceModel describes the data source data model.
type ProjectDataSourceModel struct {
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	OrganisationId types.String `tfsdk:"organisation_id"`
	Archived       types.Bool   `tfsdk:"archived"`
	ArchivedAt     types.String `tfsdk:"archived_at"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (d *ProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Project data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Project identifier",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Project name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Project description",
				Computed:            true,
			},
			"organisation_id": schema.StringAttribute{
				MarkdownDescription: "Organisation ID that owns this project",
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

func (d *ProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(map[string]interface{})
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
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

	d.client = client
	d.endpoint = endpoint
}

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", strings.TrimSuffix(d.endpoint, "/")+"/api/v1/projects", nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create request, got error: %s", err))
		return
	}

	httpReq.Header.Set("Accept", "application/json")

	// Make API call
	httpResp, err := d.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read projects, got error: %s", err))
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
	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse response, got error: %s", err))
		return
	}

	// Find project by name
	var project *Project
	for _, p := range projects {
		if p.Name == data.Name.ValueString() {
			project = &p
			break
		}
	}

	if project == nil {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("No project found with name: %s", data.Name.ValueString()))
		return
	}

	// Update model with response data
	data.Id = types.StringValue(project.Id)
	data.Name = types.StringValue(project.Name)
	data.Description = types.StringValue(project.Description)
	data.OrganisationId = types.StringValue(project.OrganisationId)
	data.Archived = types.BoolValue(project.Archived)
	data.ArchivedAt = types.StringValue(project.ArchivedAt)
	data.CreatedAt = types.StringValue(project.CreatedAt)
	data.UpdatedAt = types.StringValue(project.UpdatedAt)

	// Write logs using the tflog package
	tflog.Trace(ctx, "read a project data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
