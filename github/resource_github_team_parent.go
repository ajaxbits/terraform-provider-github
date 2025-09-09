package github

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/go-github/v66/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGithubTeamParent() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubTeamParentCreateOrUpdate,
		Read:   resourceGithubTeamParentRead,
		Update: resourceGithubTeamParentCreateOrUpdate,
		Delete: resourceGithubTeamParentDelete,
		Importer: &schema.ResourceImporter{
			State: resourceGithubTeamParentImport,
		},

		CustomizeDiff: customdiff.Sequence(
			customdiff.ComputedIf("slug", func(_ context.Context, d *schema.ResourceDiff, meta interface{}) bool {
				return d.HasChange("name")
			}),
		),

		Schema: map[string]*schema.Schema{
			"child_team_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID or slug of the child team.",
			},
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"parent_team_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The ID or slug of the parent team, if this is a nested team.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if d.Get("parent_team_id") == d.Get("parent_team_read_id") || d.Get("parent_team_id") == d.Get("parent_team_read_slug") {
						return true
					}
					return false
				},
			},
			"parent_team_read_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The id of the parent team read in Github.",
			},
			"parent_team_read_slug": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The id of the parent team read in Github.",
			},
		},
	}
}

func resourceGithubTeamParentCreateOrUpdate(d *schema.ResourceData, meta interface{}) error {
	err := checkOrganization(meta)
	if err != nil {
		return err
	}

	client := meta.(*Owner).v3client
	orgId := meta.(*Owner).id

	childTeamID := d.Get("child_team_id")
	childTeamId, err := getTeamID(childTeamID.(string), meta)
	if err != nil {
		return err
	}

	parentTeamID := d.Get("parent_team_id")
	parentTeamId, err := getTeamID(parentTeamID.(string), meta)
	if err != nil {
		return err
	}

	ctx := context.Background()

	childTeam, _, err := client.Teams.GetTeamByID(ctx, orgId, childTeamId)
	if err != nil {
		return err
	}

	newTeam := github.NewTeam{
		Name:        *childTeam.Name,
		Description: childTeam.Description,
		Privacy:     childTeam.Privacy,
	}
	newTeam.ParentTeamID = &parentTeamId
	removeParentTeam := false

	editedTeam, _, err := client.Teams.EditTeamByID(ctx, orgId, childTeamId, newTeam, removeParentTeam)
	if err != nil {
		return err
	}

	d.SetId(strconv.FormatInt(editedTeam.GetID(), 10))
	return resourceGithubTeamParentRead(d, meta)
}

func resourceGithubTeamParentRead(d *schema.ResourceData, meta interface{}) error {
	err := checkOrganization(meta)
	if err != nil {
		return err
	}

	client := meta.(*Owner).v3client
	orgId := meta.(*Owner).id

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return unconvertibleIdErr(d.Id(), err)
	}
	ctx := context.WithValue(context.Background(), ctxId, d.Id())
	if !d.IsNewResource() {
		ctx = context.WithValue(ctx, ctxEtag, d.Get("etag").(string))
	}

	team, resp, err := client.Teams.GetTeamByID(ctx, orgId, id)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return nil
			}
		}
		return err
	}

	if err = d.Set("etag", resp.Header.Get("ETag")); err != nil {
		return err
	}

	if parent := team.Parent; parent != nil {
		if err = d.Set("parent_team_id", strconv.FormatInt(team.Parent.GetID(), 10)); err != nil {
			return err
		}
		if err = d.Set("parent_team_read_id", strconv.FormatInt(team.Parent.GetID(), 10)); err != nil {
			return err
		}
		if err = d.Set("parent_team_read_slug", parent.Slug); err != nil {
			return err
		}
	} else {
		if err = d.Set("parent_team_id", ""); err != nil {
			return err
		}
		if err = d.Set("parent_team_read_id", ""); err != nil {
			return err
		}
		if err = d.Set("parent_team_read_slug", ""); err != nil {
			return err
		}
	}

	return nil
}

func resourceGithubTeamParentDelete(d *schema.ResourceData, meta interface{}) error {
	err := checkOrganization(meta)
	if err != nil {
		return err
	}

	client := meta.(*Owner).v3client
	orgId := meta.(*Owner).id

	ctx := context.Background()

	childTeamID := d.Get("child_team_id")
	childTeamId, err := getTeamID(childTeamID.(string), meta)
	if err != nil {
		return err
	}

	team, _, err := client.Teams.GetTeamByID(ctx, orgId, childTeamId)
	if err != nil {
		return err
	}

	editedTeam := github.NewTeam{
		Name:        *team.Name,
		Description: team.Description,
		Privacy:     team.Privacy,
	}
	removeParentTeam := true

	_, _, err = client.Teams.EditTeamByID(ctx, *team.Organization.ID, childTeamId, editedTeam, removeParentTeam)
	return err
}

func resourceGithubTeamParentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	childTeamId, err := getTeamID(d.Id(), meta)
	if err != nil {
		return nil, err
	}

	d.SetId(strconv.FormatInt(childTeamId, 10))

	return []*schema.ResourceData{d}, nil
}
