package resolvers

import (
	"context"
	"strconv"
	"sync"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/database"
)

type batchChangesCodeHostConnectionResolver struct {
	userID                *int32
	onlyWithoutCredential bool
	opts                  store.ListCodeHostsOpts
	limitOffset           database.LimitOffset
	store                 *store.Store

	once          sync.Once
	chs           []*btypes.CodeHost
	chsPage       []*btypes.CodeHost
	credsByIDType map[idType]graphqlbackend.BatchChangesCredentialResolver
	chsErr        error
}

var _ graphqlbackend.BatchChangesCodeHostConnectionResolver = &batchChangesCodeHostConnectionResolver{}

func (c *batchChangesCodeHostConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	chs, _, _, err := c.compute(ctx)
	if err != nil {
		return 0, err
	}
	return int32(len(chs)), err
}

func (c *batchChangesCodeHostConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	chs, _, _, err := c.compute(ctx)
	if err != nil {
		return nil, err
	}

	idx := c.limitOffset.Limit + c.limitOffset.Offset
	if idx < len(chs) {
		return graphqlutil.NextPageCursor(strconv.Itoa(idx)), nil
	}

	return graphqlutil.HasNextPage(false), nil
}

func (c *batchChangesCodeHostConnectionResolver) Nodes(ctx context.Context) ([]graphqlbackend.BatchChangesCodeHostResolver, error) {
	_, page, credsByIDType, err := c.compute(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make([]graphqlbackend.BatchChangesCodeHostResolver, len(page))
	for i, ch := range page {
		t := idType{
			externalServiceID:   ch.ExternalServiceID,
			externalServiceType: ch.ExternalServiceType,
		}
		cred := credsByIDType[t]
		nodes[i] = &batchChangesCodeHostResolver{codeHost: ch, credential: cred}
	}

	return nodes, nil
}

func (c *batchChangesCodeHostConnectionResolver) compute(ctx context.Context) (all, page []*btypes.CodeHost, credsByIDType map[idType]graphqlbackend.BatchChangesCredentialResolver, err error) {
	c.once.Do(func() {
		// Don't pass c.limitOffset here, as we want all code hosts for the totalCount anyways.
		c.chs, c.chsErr = c.store.ListCodeHosts(ctx, c.opts)
		if c.chsErr != nil {
			return
		}

		// Fetch all credentials to avoid N+1 per credential resolver.
		var userCreds []*database.UserCredential
		if c.userID != nil {
			userCreds, _, err = c.store.UserCredentials().List(ctx, database.UserCredentialsListOpts{Scope: database.UserCredentialScope{
				Domain: database.UserCredentialDomainBatches,
				UserID: *c.userID,
			}})
			if err != nil {
				c.chsErr = err
				return
			}
		}
		siteCreds, _, err := c.store.ListSiteCredentials(ctx, store.ListSiteCredentialsOpts{})
		if err != nil {
			c.chsErr = err
			return
		}

		c.credsByIDType = make(map[idType]graphqlbackend.BatchChangesCredentialResolver)
		for _, cred := range userCreds {
			t := idType{
				externalServiceID:   cred.ExternalServiceID,
				externalServiceType: cred.ExternalServiceType,
			}
			c.credsByIDType[t] = &batchChangesUserCredentialResolver{credential: cred}
		}
		for _, cred := range siteCreds {
			t := idType{
				externalServiceID:   cred.ExternalServiceID,
				externalServiceType: cred.ExternalServiceType,
			}
			if _, ok := c.credsByIDType[t]; ok {
				continue
			}
			c.credsByIDType[t] = &batchChangesSiteCredentialResolver{credential: cred}
		}

		if c.onlyWithoutCredential {
			chs := make([]*btypes.CodeHost, 0)
			for _, ch := range c.chs {
				t := idType{
					externalServiceID:   ch.ExternalServiceID,
					externalServiceType: ch.ExternalServiceType,
				}
				if _, ok := c.credsByIDType[t]; !ok {
					chs = append(chs, ch)
				}
			}
			c.chs = chs
		}

		afterIdx := c.limitOffset.Offset

		// Out of bound means page slice is empty.
		if afterIdx >= len(c.chs) {
			return
		}

		// Prepare page slice based on pagination params.
		limit := c.limitOffset.Limit
		// No limit set: page slice is all from `afterIdx` on.
		if limit <= 0 {
			c.chsPage = c.chs[afterIdx:]
			return
		}
		// If limit + afterIdx exceed slice bounds, cap to limit.
		if limit+afterIdx >= len(c.chs) {
			limit = len(c.chs) - afterIdx
		}
		c.chsPage = c.chs[afterIdx : limit+afterIdx]
	})
	return c.chs, c.chsPage, c.credsByIDType, c.chsErr
}

type idType struct {
	externalServiceID   string
	externalServiceType string
}
