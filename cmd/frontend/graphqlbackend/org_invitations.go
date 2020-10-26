package graphqlbackend

import (
	"context"
	"fmt"
	"net/url"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/envvar"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/globals"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/txemail"
	"github.com/sourcegraph/sourcegraph/internal/txemail/txtypes"
)

func getUserToInviteToOrganization(ctx context.Context, username string, orgID int32) (userToInvite *types.User, userEmailAddress string, err error) {
	userToInvite, err = db.Users.GetByUsername(ctx, username)
	if err != nil {
		return nil, "", err
	}

	if conf.CanSendEmail() {
		// Look up user's email address so we can send them an email (if needed).
		email, verified, err := db.UserEmails.GetPrimaryEmail(ctx, userToInvite.ID)
		if err != nil && !errcode.IsNotFound(err) {
			return nil, "", errors.WithMessage(err, "looking up invited user's primary email address")
		}
		if verified {
			// Completely discard unverified emails.
			userEmailAddress = email
		}
	}

	if _, err := db.OrgMembers.GetByOrgIDAndUserID(ctx, orgID, userToInvite.ID); err == nil {
		return nil, "", errors.New("user is already a member of the organization")
	} else if _, ok := err.(*db.ErrOrgMemberNotFound); !ok {
		return nil, "", err
	}
	return userToInvite, userEmailAddress, nil
}

type inviteUserToOrganizationResult struct {
	sentInvitationEmail bool
	invitationURL       string
}

func (r *inviteUserToOrganizationResult) SentInvitationEmail() bool { return r.sentInvitationEmail }
func (r *inviteUserToOrganizationResult) InvitationURL() string     { return r.invitationURL }

func (*schemaResolver) InviteUserToOrganization(ctx context.Context, args *struct {
	Organization graphql.ID
	Username     string
}) (*inviteUserToOrganizationResult, error) {
	var orgID int32
	if err := relay.UnmarshalSpec(args.Organization, &orgID); err != nil {
		return nil, err
	}
	// 🚨 SECURITY: Check that the current user is a member of the org that the user is being
	// invited to.
	if err := backend.CheckOrgAccess(ctx, orgID); err != nil {
		return nil, err
	}

	// Create the invitation.
	org, err := db.Orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	sender, err := db.Users.GetByCurrentAuthUser(ctx)
	if err != nil {
		return nil, err
	}
	recipient, recipientEmail, err := getUserToInviteToOrganization(ctx, args.Username, orgID)
	if err != nil {
		return nil, err
	}
	if _, err := db.OrgInvitations.Create(ctx, orgID, sender.ID, recipient.ID); err != nil {
		return nil, err
	}
	result := &inviteUserToOrganizationResult{
		invitationURL: globals.ExternalURL().ResolveReference(orgInvitationURL(org)).String(),
	}

	// Send a notification to the recipient. If disabled, the frontend will still show the
	// invitation link.
	if conf.CanSendEmail() && recipientEmail != "" {
		if err := sendOrgInvitationNotification(ctx, org, sender, recipientEmail); err != nil {
			return nil, errors.WithMessage(err, "sending notification to invitation recipient")
		}
		result.sentInvitationEmail = true
	}

	return result, nil
}

func (*schemaResolver) RespondToOrganizationInvitation(ctx context.Context, args *struct {
	OrganizationInvitation graphql.ID
	ResponseType           string
}) (*EmptyResponse, error) {
	currentUser, err := CurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	if currentUser == nil {
		return nil, errors.New("no current user")
	}

	id, err := unmarshalOrgInvitationID(args.OrganizationInvitation)
	if err != nil {
		return nil, err
	}

	// Convert from GraphQL enum to Go bool.
	var accept bool
	switch args.ResponseType {
	case "ACCEPT":
		accept = true
	case "REJECT":
		// noop
	default:
		return nil, fmt.Errorf("invalid OrganizationInvitationResponseType value %q", args.ResponseType)
	}

	// 🚨 SECURITY: This fails if the org invitation's recipient is not the one given (or if the
	// invitation is otherwise invalid), so we do not need to separately perform that check.
	orgID, err := db.OrgInvitations.Respond(ctx, id, currentUser.user.ID, accept)
	if err != nil {
		return nil, err
	}

	if accept {
		// The recipient accepted the invitation.
		if _, err := db.OrgMembers.Create(ctx, orgID, currentUser.user.ID); err != nil {
			return nil, err
		}
	}
	return &EmptyResponse{}, nil
}

func (*schemaResolver) ResendOrganizationInvitationNotification(ctx context.Context, args *struct {
	OrganizationInvitation graphql.ID
}) (*EmptyResponse, error) {
	orgInvitation, err := orgInvitationByID(ctx, args.OrganizationInvitation)
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: Check that the current user is a member of the org that the invite is for.
	if err := backend.CheckOrgAccess(ctx, orgInvitation.v.OrgID); err != nil {
		return nil, err
	}

	// Prevent reuse. This just prevents annoyance (abuse is prevented by the quota check in the
	// call to sendOrgInvitationNotification).
	if orgInvitation.v.RevokedAt != nil {
		return nil, errors.New("refusing to send notification for revoked invitation")
	}
	if orgInvitation.v.RespondedAt != nil {
		return nil, errors.New("refusing to send notification for invitation that was already responded to")
	}

	if !conf.CanSendEmail() {
		return nil, errors.New("unable to send notification for invitation because sending emails is not enabled")
	}

	org, err := db.Orgs.GetByID(ctx, orgInvitation.v.OrgID)
	if err != nil {
		return nil, err
	}
	sender, err := db.Users.GetByCurrentAuthUser(ctx)
	if err != nil {
		return nil, err
	}
	recipientEmail, recipientEmailVerified, err := db.UserEmails.GetPrimaryEmail(ctx, orgInvitation.v.RecipientUserID)
	if err != nil {
		return nil, err
	}
	if !recipientEmailVerified {
		return nil, errors.New("refusing to send notification because recipient has no verified email address")
	}
	if err := sendOrgInvitationNotification(ctx, org, sender, recipientEmail); err != nil {
		return nil, err
	}
	return &EmptyResponse{}, nil
}

func (*schemaResolver) RevokeOrganizationInvitation(ctx context.Context, args *struct {
	OrganizationInvitation graphql.ID
}) (*EmptyResponse, error) {
	orgInvitation, err := orgInvitationByID(ctx, args.OrganizationInvitation)
	if err != nil {
		return nil, err
	}

	// 🚨 SECURITY: Check that the current user is a member of the org that the invite is for.
	if err := backend.CheckOrgAccess(ctx, orgInvitation.v.OrgID); err != nil {
		return nil, err
	}

	if err := db.OrgInvitations.Revoke(ctx, orgInvitation.v.ID); err != nil {
		return nil, err
	}
	return &EmptyResponse{}, nil
}

func orgInvitationURL(org *types.Org) *url.URL {
	return &url.URL{Path: fmt.Sprintf("/organizations/%s/invitation", org.Name)}
}

// sendOrgInvitationNotification sends an email to the recipient of an org invitation with a link to
// respond to the invitation. Callers should check conf.CanSendEmail() if they want to return a nice
// error if sending email is not enabled.
func sendOrgInvitationNotification(ctx context.Context, org *types.Org, sender *types.User, recipientEmail string) error {
	if envvar.SourcegraphDotComMode() {
		// Basic abuse prevention for Sourcegraph.com.

		// Only allow email-verified users to send invites.
		if _, senderEmailVerified, err := db.UserEmails.GetPrimaryEmail(ctx, sender.ID); err != nil {
			return err
		} else if !senderEmailVerified {
			return errors.New("must verify your email address to invite a user to an organization")
		}

		// Check and decrement our invite quota, to prevent abuse (sending too many invites).
		//
		// There is no user invite quota for on-prem instances because we assume they can
		// trust their users to not abuse invites.
		if ok, err := db.Users.CheckAndDecrementInviteQuota(ctx, sender.ID); err != nil {
			return err
		} else if !ok {
			return errors.New("invite quota exceeded (contact support to increase the quota)")
		}
	}

	var fromName string
	if sender.DisplayName != "" {
		fromName = sender.DisplayName
	} else {
		fromName = sender.Username
	}

	return txemail.Send(ctx, txemail.Message{
		To:       []string{recipientEmail},
		Template: emailTemplates,
		Data: struct {
			FromName string
			OrgName  string
			URL      string
		}{
			FromName: fromName,
			OrgName:  org.Name,
			URL:      globals.ExternalURL().ResolveReference(orgInvitationURL(org)).String(),
		},
	})
}

var emailTemplates = txemail.MustValidate(txtypes.Templates{
	Subject: `{{.FromName}} invited you to join {{.OrgName}} on Sourcegraph`,
	Text: `
{{.FromName}} invited you to join the {{.OrgName}} organization on Sourcegraph.

To accept the invitation, follow this link:

  {{.URL}}
`,
	HTML: `
<p>
  <strong>{{.FromName}}</strong> invited you to join the
  <strong>{{.OrgName}}</strong> organization on Sourcegraph.
</p>

<p><strong><a href="{{.URL}}">Join {{.OrgName}}</a></strong></p>
`,
})
