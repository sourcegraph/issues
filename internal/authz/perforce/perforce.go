package perforce

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/inconshreveable/log15"
	jsoniter "github.com/json-iterator/go"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"

	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/extsvc/perforce"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

var _ authz.Provider = (*Provider)(nil)

// Provider implements authz.Provider for Perforce depot permissions.
type Provider struct {
	urn      string
	codeHost *extsvc.CodeHost

	host     string
	user     string
	password string
}

// NewProvider returns a new Perforce authorization provider that uses the given
// host, user and password to talk to a Perforce Server that is the source of
// truth for permissions. It assumes emails of Sourcegraph accounts match 1-1
// with emails of Perforce Server users.
func NewProvider(urn, host, user, password string) *Provider {
	baseURL, _ := url.Parse(host)
	return &Provider{
		urn:      urn,
		codeHost: extsvc.NewCodeHost(baseURL, extsvc.TypePerforce),
		host:     host,
		user:     user,
		password: password,
	}
}

// FetchAccount uses given user's verified emails to match users on the Perforce
// Server. It returns when any of the verified email has matched and the match
// result is not deterministic.
func (p *Provider) FetchAccount(ctx context.Context, user *types.User, _ []*extsvc.Account, verifiedEmails []string) (_ *extsvc.Account, err error) {
	if user == nil {
		return nil, nil
	}

	tr, ctx := trace.New(ctx, "perforce.authz.provider.FetchAccount", "")
	defer func() {
		tr.LogFields(
			otlog.String("user.name", user.Username),
			otlog.Int32("user.id", user.ID),
		)

		if err != nil {
			tr.SetError(err)
		}

		tr.Finish()
	}()

	emailSet := make(map[string]struct{}, len(verifiedEmails))
	for _, email := range verifiedEmails {
		emailSet[email] = struct{}{}
	}

	rc, _, err := gitserver.DefaultClient.P4Exec(ctx, p.host, p.user, p.password, "users")
	if err != nil {
		return nil, errors.Wrap(err, "list users")
	}
	defer func() { _ = rc.Close() }()

	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		// e.g. alice <alice@example.com> (Alice) accessed 2020/12/04
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) < 2 {
			continue
		}
		username := fields[0]                  // e.g. alice
		email := strings.Trim(fields[1], "<>") // e.g. alice@example.com

		if _, ok := emailSet[email]; ok {
			accountData, err := jsoniter.Marshal(
				perforce.AccountData{
					Username: username,
					Email:    email,
				},
			)
			if err != nil {
				return nil, err
			}

			return &extsvc.Account{
				UserID: user.ID,
				AccountSpec: extsvc.AccountSpec{
					ServiceType: p.codeHost.ServiceType,
					ServiceID:   p.codeHost.ServiceID,
					AccountID:   email,
				},
				AccountData: extsvc.AccountData{
					Data: (*json.RawMessage)(&accountData),
				},
			}, nil
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "scanner.Err")
	}

	// Drain remaining body
	_, _ = io.Copy(ioutil.Discard, rc)
	return nil, nil
}

// FetchUserPerms returns a list of depot prefixes that the given user has
// access to on the Perforce Server.
func (p *Provider) FetchUserPerms(ctx context.Context, account *extsvc.Account) (*authz.ExternalUserPermissions, error) {
	user, err := perforce.GetExternalAccountData(&account.AccountData)
	if err != nil {
		return nil, errors.Wrap(err, "get external account data")
	}

	rc, _, err := gitserver.DefaultClient.P4Exec(ctx, p.host, p.user, p.password, "protects", "-u", user.Username)
	if err != nil {
		return nil, errors.Wrap(err, "list ACLs by user")
	}
	defer func() { _ = rc.Close() }()

	var includePrefixes, excludePrefixes []extsvc.RepoID
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()

		// Trim comments
		i := strings.Index(line, "##")
		if i > -1 {
			line = line[:i]
		}

		// e.g. read user alice * //Sourcegraph/...
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) < 5 {
			continue
		}
		level := fields[0]                               // e.g. read
		depotPrefix := strings.TrimRight(fields[4], ".") // e.g. //Sourcegraph/

		// Rule that starts with a "-" in depot prefix means exclusion (i.e. revoke access)
		if strings.HasPrefix(depotPrefix, "-") {
			_, canRevokeReadAccess := map[string]struct{}{
				"list":   {},
				"read":   {},
				"=read":  {},
				"open":   {},
				"write":  {},
				"review": {},
				"owner":  {},
				"admin":  {},
				"super":  {},
			}[level]
			if !canRevokeReadAccess {
				continue
			}

			for i, prefix := range includePrefixes {
				if !strings.HasPrefix(depotPrefix[1:], string(prefix)) {
					continue
				} else if depotPrefix[1:] == string(prefix) {
					includePrefixes = append(includePrefixes[:i], includePrefixes[i+1:]...)
					break
				}

				excludePrefixes = append(excludePrefixes, extsvc.RepoID(depotPrefix[1:]))
				break
			}

		} else {
			_, canGrantReadAccess := map[string]struct{}{
				"read":   {},
				"=read":  {},
				"open":   {},
				"=open":  {},
				"write":  {},
				"=write": {},
				"review": {},
				"owner":  {},
				"admin":  {},
				"super":  {},
			}[level]
			if !canGrantReadAccess {
				continue
			}

			includePrefixes = append(includePrefixes, extsvc.RepoID(depotPrefix))
		}
	}

	// As per interface definition for this method, implementation should return
	// partial but valid results even when something went wrong.
	return &authz.ExternalUserPermissions{
		IncludePrefixes: includePrefixes,
		ExcludePrefixes: excludePrefixes,
	}, errors.Wrap(scanner.Err(), "scanner.Err")
}

// FetchRepoPerms returns a list of users that have access to the given
// repository on the Perforce Server.
func (p *Provider) FetchRepoPerms(ctx context.Context, repo *extsvc.Repository) ([]extsvc.AccountID, error) {
	rc, _, err := gitserver.DefaultClient.P4Exec(ctx, p.host, p.user, p.password, "protects", "-a", repo.ID)
	if err != nil {
		return nil, errors.Wrap(err, "list ACLs by depot")
	}
	defer func() { _ = rc.Close() }()

	var cachedUserEmails map[string]string
	getAllUserEmails := func() (map[string]string, error) {
		if cachedUserEmails != nil {
			return cachedUserEmails, nil
		}

		userEmails := make(map[string]string)
		rc, _, err := gitserver.DefaultClient.P4Exec(ctx, p.host, p.user, p.password, "users")
		if err != nil {
			return nil, errors.Wrap(err, "list users")
		}
		defer func() { _ = rc.Close() }()

		scanner := bufio.NewScanner(rc)
		for scanner.Scan() {
			// e.g. alice <alice@example.com> (Alice) accessed 2020/12/04
			fields := strings.Split(scanner.Text(), " ")
			if len(fields) < 2 {
				continue
			}
			username := fields[0]                  // e.g. alice
			email := strings.Trim(fields[1], "<>") // e.g. alice@example.com

			userEmails[username] = email
		}
		if err = scanner.Err(); err != nil {
			return nil, errors.Wrap(err, "scanner.Err")
		}
		fmt.Println("getAllUsers", userEmails)

		cachedUserEmails = userEmails
		return userEmails, nil
	}
	getAllUsers := func() ([]string, error) {
		userEmails, err := getAllUserEmails()
		if err != nil {
			return nil, errors.Wrap(err, "get all user emails")
		}

		users := make([]string, 0, len(userEmails))
		for name := range userEmails {
			users = append(users, name)
		}
		return users, nil
	}
	getGroupMembers := func(group string) ([]string, error) {
		rc, _, err := gitserver.DefaultClient.P4Exec(ctx, p.host, p.user, p.password, "group", "-o", group)
		if err != nil {
			return nil, errors.Wrap(err, "list group members")
		}
		defer func() { _ = rc.Close() }()

		var members []string
		startScan := false
		scanner := bufio.NewScanner(rc)
		for scanner.Scan() {
			line := scanner.Text()

			// Only start scan when we encounter the "Users:" line
			if !startScan {
				if strings.HasPrefix(line, "Users:") {
					startScan = true
				}
				continue
			}

			// Lines for users always start with a tab "\t"
			if !strings.HasPrefix(line, "\t") {
				break
			}

			members = append(members, strings.TrimSpace(line))
		}
		if err = scanner.Err(); err != nil {
			return nil, errors.Wrap(err, "scanner.Err")
		}

		// Drain remaining body
		_, _ = io.Copy(ioutil.Discard, rc)
		fmt.Println("getGroupMembers", group, members)
		return members, nil
	}

	users := make(map[string]struct{})
	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()

		// Trim comments
		i := strings.Index(line, "##")
		if i > -1 {
			line = line[:i]
		}

		// e.g. write user alice * //Sourcegraph/...
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) < 5 {
			continue
		}
		level := fields[0]                               // e.g. read
		typ := fields[1]                                 // e.g. user
		name := fields[2]                                // e.g. alice
		depotPrefix := strings.TrimRight(fields[4], ".") // e.g. //Sourcegraph/

		// Rule that starts with a "-" in depot prefix means exclusion (i.e. revoke access)
		if strings.HasPrefix(depotPrefix, "-") {
			_, canRevokeReadAccess := map[string]struct{}{
				"list":   {},
				"read":   {},
				"=read":  {},
				"open":   {},
				"write":  {},
				"review": {},
				"owner":  {},
				"admin":  {},
				"super":  {},
			}[level]
			if !canRevokeReadAccess {
				continue
			}

			switch typ {
			case "user":
				if name == "*" {
					users = make(map[string]struct{})
				} else {
					delete(users, name)
				}
			case "group":
				members, err := getGroupMembers(name)
				if err != nil {
					return nil, errors.Wrapf(err, "list members of group %q", name)
				}
				for _, member := range members {
					delete(users, member)
				}

			default:
				log15.Warn("authz.perforce.Provider.FetchRepoPerms.unrecognizedType", "type", typ)
			}

		} else {
			_, canGrantReadAccess := map[string]struct{}{
				"read":   {},
				"=read":  {},
				"open":   {},
				"=open":  {},
				"write":  {},
				"=write": {},
				"review": {},
				"owner":  {},
				"admin":  {},
				"super":  {},
			}[level]
			if !canGrantReadAccess {
				continue
			}

			switch typ {
			case "user":
				if name == "*" {
					all, err := getAllUsers()
					if err != nil {
						return nil, errors.Wrap(err, "list all users")
					}
					for _, user := range all {
						users[user] = struct{}{}
					}
				} else {
					users[name] = struct{}{}
				}
			case "group":
				members, err := getGroupMembers(name)
				if err != nil {
					return nil, errors.Wrapf(err, "list members of group %q", name)
				}
				for _, member := range members {
					users[member] = struct{}{}
				}

			default:
				log15.Warn("authz.perforce.Provider.FetchRepoPerms.unrecognizedType", "type", typ)
			}
		}

	}
	if err = scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "scanner.Err")
	}

	userEmails, err := getAllUserEmails()
	if err != nil {
		return nil, errors.Wrap(err, "get all user emails")
	}
	extIDs := make([]extsvc.AccountID, 0, len(users))
	for user := range users {
		email, ok := userEmails[user]
		if !ok {
			continue
		}
		extIDs = append(extIDs, extsvc.AccountID(email))
	}
	fmt.Println("extIDs", extIDs)
	return extIDs, nil
}

func (p *Provider) ServiceType() string {
	return p.codeHost.ServiceType
}

func (p *Provider) ServiceID() string {
	return p.codeHost.ServiceID
}

func (p *Provider) URN() string {
	return p.urn
}

func (p *Provider) Validate() (problems []string) {
	// Validate the user has "super" access with "-u" option, see https://www.perforce.com/perforce/r12.1/manuals/cmdref/protects.html
	rc, _, err := gitserver.DefaultClient.P4Exec(context.Background(), p.host, p.user, p.password, "protects", "-u", p.user)
	if err == nil {
		_ = rc.Close()
		return nil
	}

	if strings.Contains(err.Error(), "You don't have permission for this operation.") {
		return []string{"the user does not have super access"}
	}
	return []string{"validate user access level: " + err.Error()}
}
