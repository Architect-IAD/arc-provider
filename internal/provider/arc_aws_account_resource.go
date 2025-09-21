// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	orgstypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AccountResource{}
var _ resource.ResourceWithImportState = &AccountResource{}

func NewAccountResource() resource.Resource {
	return &AccountResource{}
}

// AccountResource defines the resource implementation.
type AccountResource struct {
	orgs *organizations.Client
}

// AccountResourceModel describes the resource data model.
type AccountResourceModel struct {
	ID           types.String `tfsdk:"id"`
	AccountID    types.String `tfsdk:"account_id"`
	ClosedUnitID types.String `tfsdk:"closed_unit_id"`
	UnitID       types.String `tfsdk:"unit_id"`
	Email        types.String `tfsdk:"email"`
	Name         types.String `tfsdk:"name"`
}

func (r *AccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_account"
}

func (r *AccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Example resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"account_id": schema.StringAttribute{
				Computed: true,
			},
			"closed_unit_id": schema.StringAttribute{
				MarkdownDescription: "closed unit id",
				Required:            true,
				Optional:            false,
			},
			"unit_id": schema.StringAttribute{
				MarkdownDescription: "unit id",
				Required:            true,
				Optional:            false,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Account email",
				Required:            true,
				Optional:            false,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Account name",
				Required:            true,
				Optional:            false,
			},
		},
	}
}

func (r *AccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if c, ok := req.ProviderData.(*organizations.Client); ok && c != nil {
		r.orgs = c
		tflog.Debug(ctx, "configured AccountResource with *organizations.Client (direct)")
		return
	}

	resp.Diagnostics.AddError(
		"Unexpected Provider Configuration",
		fmt.Sprintf("Expected *organizations.Client, an orgsGetter, or a wrapper with Orgs/Organizations *organizations.Client; got %T", req.ProviderData),
	)
}

func (r *AccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccountResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	email := plan.Email.ValueString()
	closedUnitID := plan.ClosedUnitID.ValueString()
	name := plan.Name.ValueString()
	unitID := plan.UnitID.ValueString()

	account, findAccError := findAccountByEmail(ctx, r.orgs, email)

	if findAccError != nil {
		resp.Diagnostics.AddError("Issue finding account", findAccError.Error())
		return
	}

	if account != nil {
		if account.Status == "SUSPENDED" {
			resp.Diagnostics.AddError("An account was found, however it is pending closure.", "Please reopen the account or wait for aws to delete it.")
			return
		}

		out, listParentsError := r.orgs.ListParents(ctx, &organizations.ListParentsInput{
			ChildId: aws.String(*account.Id),
		})

		if listParentsError != nil {
			resp.Diagnostics.AddError("Issue listing parents", listParentsError.Error())
			return
		}

		if len(out.Parents) != 1 {
			resp.Diagnostics.AddError("The account has too many or few parents.", "Possibly modified outside of terraform")
			return
		}

		if closedUnitID == *out.Parents[0].Id {
			_, moveAccountError := r.orgs.MoveAccount(ctx, &organizations.MoveAccountInput{
				AccountId:           aws.String(*account.Id),
				SourceParentId:      aws.String(closedUnitID),
				DestinationParentId: aws.String(unitID),
			})

			if moveAccountError != nil {
				resp.Diagnostics.AddError("Error moving the account", moveAccountError.Error())
				return
			}

			plan.AccountID = types.StringValue(*account.Id)
		} else {
			resp.Diagnostics.AddWarning("An account was already found in the same organizational unit.", "If this account was not part of a timeout issue you may have duplicate account emails.")
			return
		}
	} else {
		newAccount, err := r.orgs.CreateAccount(ctx, &organizations.CreateAccountInput{
			AccountName: aws.String(name),
			Email:       aws.String(email),
		})

		if err != nil {
			resp.Diagnostics.AddError("Issue creating account", err.Error())
			return
		}

		result := waitForAccountCreation(ctx, r.orgs, *newAccount.CreateAccountStatus.Id)

		if !result {
			resp.Diagnostics.AddError("Error while waiting for account creation.", "Timeout issue. You can retry again.")
			return
		}

		plan.AccountID = types.StringValue(*newAccount.CreateAccountStatus.AccountId)

		_, moveAccountError := r.orgs.MoveAccount(ctx, &organizations.MoveAccountInput{
			AccountId:           aws.String(*newAccount.CreateAccountStatus.AccountId),
			DestinationParentId: aws.String(unitID),
		})

		if moveAccountError != nil {
			resp.Diagnostics.AddError("Error moving the account", moveAccountError.Error())
			return
		}
	}

	plan.ID = types.StringValue(fmt.Sprintf("arcorg:%s", plan.AccountID.ValueString()))

	tflog.Trace(ctx, "created account resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccountResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	account, err := findAccountByEmail(ctx, r.orgs, data.Email.ValueString())

	if err != nil {
		resp.Diagnostics.AddError("Issue finding account", err.Error())
		return
	}

	if account == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.AccountID = types.StringPointerValue(account.Id)
	data.ID = types.StringValue(fmt.Sprintf("arcorg:%s", account.Id))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Email != state.Email || plan.Name != state.Name || state.UnitID != plan.UnitID || state.ClosedUnitID != plan.ClosedUnitID {
		resp.Diagnostics.AddError("Cannot Modify Account after creation", "Destroy this resource and re-create it")
		return
	}
}

func (r *AccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, moveAccountError := r.orgs.MoveAccount(ctx, &organizations.MoveAccountInput{
		AccountId:           aws.String(state.AccountID.ValueString()),
		SourceParentId:      aws.String(state.UnitID.ValueString()),
		DestinationParentId: aws.String(state.ClosedUnitID.ValueString()),
	})

	if moveAccountError != nil {
		resp.Diagnostics.AddError("Error moving the account", moveAccountError.Error())
		return
	}
}

func (r *AccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func findAccountByEmail(ctx context.Context, c *organizations.Client, email string) (*orgstypes.Account, error) {
	var next *string
	for {
		out, err := c.ListAccounts(ctx, &organizations.ListAccountsInput{NextToken: next})
		if err != nil {
			return nil, err
		}
		for _, a := range out.Accounts {
			if aws.ToString(a.Email) == email {
				return &a, nil
			}
		}
		if out.NextToken == nil {
			break
		}
		next = out.NextToken
	}
	return nil, nil
}

func waitForAccountCreation(ctx context.Context, c *organizations.Client, reqID string) bool {
	for i := 0; i < 60; i++ {
		time.Sleep(10 * time.Second)
		desc, err := c.DescribeCreateAccountStatus(ctx, &organizations.DescribeCreateAccountStatusInput{
			CreateAccountRequestId: aws.String(reqID),
		})
		if err != nil {
			continue
		}
		st := desc.CreateAccountStatus
		if st == nil {
			continue
		}
		switch st.State {
		case orgstypes.CreateAccountStateSucceeded:
			return true
		case orgstypes.CreateAccountStateFailed:
			return false
		}
	}
	return false
}
