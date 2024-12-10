package nginx

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
)

type NginxProvider struct {
	version string
}

// NginxProviderModel describes the provider data model.
type NginxProviderModel struct {
	Host     types.String `tfsdk:"host"`
	User     types.String `tfsdk:"user"`
	Password types.String `tfsdk:"password"`
}

type Client struct {
	SSHClient *ssh.Client
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return nil

	}
}

func (p *NginxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nginx"
	resp.Version = p.version
}

func (p *NginxProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider for managing NGINX configurations via SSH.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "The hostname or IP address of the NGINX server.",
				Required:            true,
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "The SSH username to connect to the NGINX server.",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The SSH password to connect to the NGINX server.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *NginxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data NginxProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := data.Host.ValueString()
	user := data.User.ValueString()
	password := data.Password.ValueString()

	log.Printf("[DEBUG] Initializing client for Host: %s, User: %s", host, user)

	client, err := NewClient(host, user, password)
	if err != nil {
		resp.Diagnostics.AddError("Client Initialization Error", fmt.Sprintf("Unable to initialize SSH client: %s", err))
		return
	}

	log.Println("[DEBUG] Client initialized successfully.")
	resp.ResourceData = client
}

// NewClient creates a new SSH client
func NewClient(host, user, password string) (*Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Do not use in production
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	return &Client{SSHClient: conn}, nil
}

// RunCommand executes a command on the remote server
func (c *Client) RunCommand(command string) (string, error) {
	session, err := c.SSHClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	stdout, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to execute command '%s': %w", command, err)
	}

	return string(stdout), nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	return c.SSHClient.Close()
}

func (p *NginxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Add resources here
	}
}
