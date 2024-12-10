package nginx

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NginxConfResource{}
var _ resource.ResourceWithImportState = &NginxConfResource{}

func NewNginxConfResource() resource.Resource {
	return &NginxConfResource{}
}

// NginxConfResource defines the resource implementation.
type NginxConfResource struct {
	client *Client
}

// NginxConfResourceModel describes the resource data model.
type NginxConfResourceModel struct {
	ServerName types.String `tfsdk:"server_name"`
	ListenPort types.Int64  `tfsdk:"listen_port"`
	Root       types.String `tfsdk:"root"`
	Path       types.String `tfsdk:"path"`
	Content    types.String `tfsdk:"content"`
	Id         types.String `tfsdk:"id"`
}

func (r *NginxConfResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nginx_conf"
}

func (r *NginxConfResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an NGINX configuration file.",
		Attributes: map[string]schema.Attribute{
			"server_name": schema.StringAttribute{
				MarkdownDescription: "The server name for the NGINX configuration.",
				Required:            true,
			},
			"listen_port": schema.Int64Attribute{
				MarkdownDescription: "The port number to listen on.",
				Required:            true,
			},
			"root": schema.StringAttribute{
				MarkdownDescription: "The root directory for the NGINX server.",
				Required:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "Path to the nginx.conf file.",
				Required:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "Content of the nginx.conf file.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource ID, which is the path to the nginx.conf file.",
			},
		},
	}

}

func (r *NginxConfResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *NginxConfResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NginxConfResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	configContent := fmt.Sprintf(`server {
	listen %d;
	server_name %s;

	root %s;
	index index.html;

	location / {
		try_files $uri $uri/ =404;
	}
}`, data.ListenPort.ValueInt64(), data.ServerName.ValueString(), data.Root.ValueString())

	tempFilePath := "/tmp/nginx_temp.conf"
	uploadCommand := fmt.Sprintf("echo '%s' > %s", shellEscape(configContent), tempFilePath)
	if _, err := r.client.RunCommand(uploadCommand); err != nil {
		resp.Diagnostics.AddError("Error uploading configuration", fmt.Sprintf("Failed to upload configuration to %s: %v", tempFilePath, err))
		return
	}

	moveCommand := fmt.Sprintf("sudo mv %s %s", tempFilePath, data.Path.ValueString())
	if _, err := r.client.RunCommand(moveCommand); err != nil {
		resp.Diagnostics.AddError("Error moving configuration", fmt.Sprintf("Failed to move configuration to %s: %v", data.Path.ValueString(), err))
		return
	}
	//	r.client.ReloadNginx()

	data.Id = types.StringValue(data.Path.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NginxConfResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NginxConfResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	command := fmt.Sprintf("cat %s", data.Path.ValueString())
	stdout, err := r.client.RunCommand(command)
	if err != nil {
		resp.Diagnostics.AddError("Error reading configuration", fmt.Sprintf("Failed to read configuration at %s: %v", data.Path.ValueString(), err))
		return
	}

	data.Content = types.StringValue(stdout)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NginxConfResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NginxConfResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	configContent := data.Content.ValueString()
	escapedContent := shellEscape(configContent)
	command := fmt.Sprintf("echo '%s' > %s", escapedContent, data.Path.ValueString())
	if _, err := r.client.RunCommand(command); err != nil {
		resp.Diagnostics.AddError("Error updating configuration", fmt.Sprintf("Failed to update configuration at %s: %v", data.Path.ValueString(), err))
		return
	}
	//	r.client.ReloadNginx()
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NginxConfResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NginxConfResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	command := fmt.Sprintf("rm -f %s", data.Path.ValueString())
	if _, err := r.client.RunCommand(command); err != nil {
		resp.Diagnostics.AddError("Error deleting configuration", fmt.Sprintf("Failed to delete configuration at %s: %v", data.Path.ValueString(), err))
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *NginxConfResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func shellEscape(content string) string {
	return strings.ReplaceAll(content, "'", "'\\''")
}
