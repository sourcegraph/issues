package resolvers

import (
	"context"
	"database/sql"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	cm "github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codemonitors/email"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

// NewResolver returns a new Resolver that uses the given database
func NewResolver(db dbutil.DB) graphqlbackend.CodeMonitorsResolver {
	return &Resolver{store: cm.NewStore(db)}
}

// newResolverWithClock is used in tests to set the clock manually.
func newResolverWithClock(db dbutil.DB, clock func() time.Time) graphqlbackend.CodeMonitorsResolver {
	return &Resolver{store: cm.NewStoreWithClock(db, clock)}
}

type Resolver struct {
	store *cm.Store
}

func (r *Resolver) Now() time.Time {
	return r.store.Now()
}

func (r *Resolver) NodeResolvers() map[string]graphqlbackend.NodeByIDFunc {
	return map[string]graphqlbackend.NodeByIDFunc{
		MonitorKind: func(ctx context.Context, id graphql.ID) (graphqlbackend.Node, error) {
			return r.MonitorByID(ctx, id)
		},
		// TODO: These kinds are currently not implemented, but need a node resolver.
		// monitorTriggerQueryKind: func(ctx context.Context, id graphql.ID) (graphqlbackend.Node, error) {
		// 	return r.MonitorTriggerQueryByID(ctx, id)
		// },
		// monitorTriggerEventKind: func(ctx context.Context, id graphql.ID) (graphqlbackend.Node, error) {
		// 	return r.MonitorTriggerEventByID(ctx, id)
		// },
		// monitorActionEmailKind: func(ctx context.Context, id graphql.ID) (graphqlbackend.Node, error) {
		// 	return r.MonitorActionEmailByID(ctx, id)
		// },
		// monitorActionEventKind: func(ctx context.Context, id graphql.ID) (graphqlbackend.Node, error) {
		// 	return r.MonitorActionEventByID(ctx, id)
		// },
	}
}

func (r *Resolver) Monitors(ctx context.Context, userID int32, args *graphqlbackend.ListMonitorsArgs) (graphqlbackend.MonitorConnectionResolver, error) {
	// Request one extra to determine if there are more pages
	newArgs := *args
	newArgs.First += 1

	ms, err := r.store.Monitors(ctx, userID, &newArgs)
	if err != nil {
		return nil, err
	}

	totalCount, err := r.store.TotalCountMonitors(ctx, userID)
	if err != nil {
		return nil, err
	}

	hasNextPage := false
	if len(ms) == int(args.First)+1 {
		hasNextPage = true
		ms = ms[:len(ms)-1]
	}

	mrs := make([]graphqlbackend.MonitorResolver, 0, len(ms))
	for _, m := range ms {
		mrs = append(mrs, &monitor{
			Resolver: r,
			Monitor:  m,
		})
	}

	return &monitorConnection{Resolver: r, monitors: mrs, totalCount: totalCount, hasNextPage: hasNextPage}, nil
}

func (r *Resolver) MonitorByID(ctx context.Context, id graphql.ID) (m graphqlbackend.MonitorResolver, err error) {
	err = r.isAllowedToEdit(ctx, id)
	if err != nil {
		return nil, err
	}
	var monitorID int64
	err = relay.UnmarshalSpec(id, &monitorID)
	if err != nil {
		return nil, err
	}
	var mo *cm.Monitor
	mo, err = r.store.MonitorByIDInt64(ctx, monitorID)
	if err != nil {
		return nil, err
	}
	return &monitor{
		Resolver: r,
		Monitor:  mo,
	}, nil
}

func (r *Resolver) CreateCodeMonitor(ctx context.Context, args *graphqlbackend.CreateCodeMonitorArgs) (m graphqlbackend.MonitorResolver, err error) {
	err = r.isAllowedToCreate(ctx, args.Monitor.Namespace)
	if err != nil {
		return nil, err
	}
	var mo *cm.Monitor
	mo, err = r.store.CreateCodeMonitor(ctx, args)
	if err != nil {
		return nil, err
	}
	return &monitor{
		Resolver: r,
		Monitor:  mo,
	}, nil
}

func (r *Resolver) ToggleCodeMonitor(ctx context.Context, args *graphqlbackend.ToggleCodeMonitorArgs) (mr graphqlbackend.MonitorResolver, err error) {
	err = r.isAllowedToEdit(ctx, args.Id)
	if err != nil {
		return nil, errors.Errorf("ToggleCodeMonitor: %w", err)
	}
	var mo *cm.Monitor
	mo, err = r.store.ToggleMonitor(ctx, args)
	if err != nil {
		return nil, err
	}
	return &monitor{r, mo}, nil
}

func (r *Resolver) DeleteCodeMonitor(ctx context.Context, args *graphqlbackend.DeleteCodeMonitorArgs) (*graphqlbackend.EmptyResponse, error) {
	err := r.isAllowedToEdit(ctx, args.Id)
	if err != nil {
		return nil, errors.Errorf("DeleteCodeMonitor: %w", err)
	}
	err = r.store.DeleteMonitor(ctx, args)
	if err != nil {
		return nil, err
	}
	return &graphqlbackend.EmptyResponse{}, nil
}

func (r *Resolver) UpdateCodeMonitor(ctx context.Context, args *graphqlbackend.UpdateCodeMonitorArgs) (m graphqlbackend.MonitorResolver, err error) {
	err = r.isAllowedToEdit(ctx, args.Monitor.Id)
	if err != nil {
		return nil, errors.Errorf("UpdateCodeMonitor: %w", err)
	}

	var monitorID int64
	err = relay.UnmarshalSpec(args.Monitor.Id, &monitorID)
	if err != nil {
		return nil, err
	}

	// Get all action IDs of the monitor.
	actionIDs, err := r.actionIDsForMonitorIDInt64(ctx, monitorID)
	if err != nil {
		return nil, err
	}

	toCreate, toDelete, err := splitActionIDs(ctx, args, actionIDs)
	if len(toDelete) == len(actionIDs) {
		return nil, errors.Errorf("you tried to delete all actions, but every monitor must be connected to at least 1 action")
	}

	// Run all queries within a transaction.
	var tx *Resolver
	tx, err = r.transact(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = tx.store.Done(err) }()

	err = tx.store.DeleteActionsInt64(ctx, toDelete, monitorID)
	if err != nil {
		return nil, err
	}
	err = tx.store.CreateActions(ctx, toCreate, monitorID)
	if err != nil {
		return nil, err
	}
	m, err = tx.updateCodeMonitor(ctx, args)
	if err != nil {
		return nil, err
	}
	// Hydrate monitor with Resolver.
	m.(*monitor).Resolver = r
	return m, nil
}

// ResetTriggerQueryTimestamps is a convenience function which resets the
// timestamps `next_run` and `last_result` with the purpose to trigger associated
// actions (emails, webhooks) immediately. This is useful during development and
// troubleshooting. Only site admins can call this functions.
func (r *Resolver) ResetTriggerQueryTimestamps(ctx context.Context, args *graphqlbackend.ResetTriggerQueryTimestampsArgs) (*graphqlbackend.EmptyResponse, error) {
	err := backend.CheckCurrentUserIsSiteAdmin(ctx, r.store.Handle().DB())
	if err != nil {
		return nil, err
	}
	var queryIDInt64 int64
	err = relay.UnmarshalSpec(args.Id, &queryIDInt64)
	if err != nil {
		return nil, err
	}
	err = r.store.ResetTriggerQueryTimestamps(ctx, queryIDInt64)
	if err != nil {
		return nil, err
	}
	return &graphqlbackend.EmptyResponse{}, nil
}

func (r *Resolver) TriggerTestEmailAction(ctx context.Context, args *graphqlbackend.TriggerTestEmailActionArgs) (*graphqlbackend.EmptyResponse, error) {
	err := r.isAllowedToCreate(ctx, args.Namespace)
	if err != nil {
		return nil, err
	}

	for _, recipient := range args.Email.Recipients {
		if err := sendTestEmail(ctx, recipient, args.Description); err != nil {
			return nil, err
		}
	}

	return &graphqlbackend.EmptyResponse{}, nil
}

func sendTestEmail(ctx context.Context, recipient graphql.ID, description string) error {
	var (
		userID int32
		orgID  int32
	)
	err := graphqlbackend.UnmarshalNamespaceID(recipient, &userID, &orgID)
	if err != nil {
		return err
	}
	// TODO: Send test email to org members.
	if orgID != 0 {
		return nil
	}
	data := email.NewTestTemplateDataForNewSearchResults(ctx, description)
	return email.SendEmailForNewSearchResult(ctx, userID, data)
}

func (r *Resolver) actionIDsForMonitorIDInt64(ctx context.Context, monitorID int64) (actionIDs []graphql.ID, err error) {
	limit := 50
	var (
		ids   []graphql.ID
		after *string
		q     *sqlf.Query
	)
	// Paging.
	for {
		q, err = r.store.ReadActionEmailQuery(ctx, monitorID, &graphqlbackend.ListActionArgs{
			First: int32(limit),
			After: after,
		})
		if err != nil {
			return nil, err
		}
		es, cur, err := r.actionIDsForMonitorIDINT64SinglePage(ctx, q, limit)
		if err != nil {
			return nil, err
		}
		ids = append(ids, es...)
		if cur == nil {
			break
		}
		after = cur
	}
	return ids, nil
}

func (r *Resolver) actionIDsForMonitorIDINT64SinglePage(ctx context.Context, q *sqlf.Query, limit int) (ids []graphql.ID, cursor *string, err error) {
	var rows *sql.Rows
	rows, err = r.store.Query(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var es []*cm.MonitorEmail
	es, err = cm.ScanEmails(rows)
	if err != nil {
		return nil, nil, err
	}
	ids = make([]graphql.ID, 0, len(es))
	for _, e := range es {
		ids = append(ids, (&monitorEmail{MonitorEmail: e}).ID())
	}

	// Set the cursor if the result size equals limit.
	if len(ids) == limit {
		stringID := string(ids[len(ids)-1])
		cursor = &stringID
	}
	return ids, cursor, nil
}

// splitActionIDs splits actions into three buckets: create, delete and update.
// Note: args is mutated. After splitActionIDs, args only contains actions to be updated.
func splitActionIDs(ctx context.Context, args *graphqlbackend.UpdateCodeMonitorArgs, actionIDs []graphql.ID) (toCreate []*graphqlbackend.CreateActionArgs, toDelete []int64, err error) {
	aMap := make(map[graphql.ID]struct{}, len(actionIDs))
	for _, id := range actionIDs {
		aMap[id] = struct{}{}
	}
	var toUpdateActions []*graphqlbackend.EditActionArgs
	for _, a := range args.Actions {
		if a.Email.Id == nil {
			toCreate = append(toCreate, &graphqlbackend.CreateActionArgs{Email: a.Email.Update})
			continue
		}
		if _, ok := aMap[*a.Email.Id]; !ok {
			return nil, nil, errors.Errorf("unknown ID=%s for action", *a.Email.Id)
		}
		toUpdateActions = append(toUpdateActions, a)
		delete(aMap, *a.Email.Id)
	}
	var actionID int64
	for k := range aMap {
		err = relay.UnmarshalSpec(k, &actionID)
		if err != nil {
			return nil, nil, err
		}
		toDelete = append(toDelete, actionID)
	}
	args.Actions = toUpdateActions
	return toCreate, toDelete, nil
}

func (r *Resolver) updateCodeMonitor(ctx context.Context, args *graphqlbackend.UpdateCodeMonitorArgs) (m graphqlbackend.MonitorResolver, err error) {
	// Update monitor.
	var mo *cm.Monitor
	mo, err = r.store.UpdateMonitor(ctx, args)
	if err != nil {
		return nil, err
	}
	// Update trigger.
	err = r.store.UpdateTriggerQuery(ctx, args)
	if err != nil {
		return nil, err
	}
	// Update actions.
	if len(args.Actions) == 0 {
		return &monitor{
			Resolver: r,
			Monitor:  mo,
		}, nil
	}
	var emailID int64
	var e *cm.MonitorEmail
	for i, action := range args.Actions {
		if action.Email == nil {
			return nil, errors.Errorf("missing email object for action %d", i)
		}
		err = relay.UnmarshalSpec(*action.Email.Id, &emailID)
		if err != nil {
			return nil, err
		}
		err = r.store.DeleteRecipients(ctx, emailID)
		if err != nil {
			return nil, err
		}
		e, err = r.store.UpdateActionEmail(ctx, mo.ID, action)
		if err != nil {
			return nil, err
		}
		err = r.store.CreateRecipients(ctx, action.Email.Update.Recipients, e.Id)
		if err != nil {
			return nil, err
		}
	}
	return &monitor{
		Resolver: r,
		Monitor:  mo,
	}, nil
}

func (r *Resolver) transact(ctx context.Context) (*Resolver, error) {
	txStore, err := r.store.Transact(ctx)
	if err != nil {
		return nil, err
	}
	return &Resolver{
		store: txStore,
	}, nil
}

// isAllowedToEdit checks whether an actor is allowed to edit a given monitor.
func (r *Resolver) isAllowedToEdit(ctx context.Context, monitorID graphql.ID) error {
	var monitorIDInt64 int64
	err := relay.UnmarshalSpec(monitorID, &monitorIDInt64)
	if err != nil {
		return err
	}
	owner, err := r.ownerForID64(ctx, monitorIDInt64)
	if err != nil {
		return err
	}
	return r.isAllowedToCreate(ctx, owner)
}

// isAllowedToCreate compares the owner of a monitor (user or org) to the actor of
// the request. A user can create a monitor if either of the following statements
// is true:
// - she is the owner
// - she is a member of the organization which is the owner of the monitor
// - she is a site-admin
func (r *Resolver) isAllowedToCreate(ctx context.Context, owner graphql.ID) error {
	var ownerInt32 int32
	err := relay.UnmarshalSpec(owner, &ownerInt32)
	if err != nil {
		return err
	}
	switch kind := relay.UnmarshalKind(owner); kind {
	case "User":
		return backend.CheckSiteAdminOrSameUser(ctx, r.store.Handle().DB(), ownerInt32)
	case "Org":
		return backend.CheckOrgAccessOrSiteAdmin(ctx, r.store.Handle().DB(), ownerInt32)
	default:
		return errors.Errorf("provided ID is not a namespace")
	}
}

func (r *Resolver) ownerForID64(ctx context.Context, monitorID int64) (owner graphql.ID, err error) {
	var (
		q      *sqlf.Query
		rows   *sql.Rows
		userID *int32
		orgID  *int32
	)
	q, err = ownerForID64Query(ctx, monitorID)
	if err != nil {
		return "", err
	}
	rows, err = r.store.Query(ctx, q)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(
			&userID,
			&orgID,
		); err != nil {
			return "", err
		}
	}
	err = rows.Close()
	if err != nil {
		return "", err
	}
	// Rows.Err will report the last error encountered by Rows.Scan.
	if err = rows.Err(); err != nil {
		return "", err
	}
	if (userID == nil && orgID == nil) || (userID != nil && orgID != nil) {
		return "", errors.Errorf("invalid owner")
	}
	if orgID != nil {
		return graphqlbackend.MarshalOrgID(*orgID), nil
	} else {
		return graphqlbackend.MarshalUserID(*userID), nil
	}
}

func ownerForID64Query(ctx context.Context, monitorID int64) (*sqlf.Query, error) {
	const ownerForId32Query = `SELECT namespace_user_id, namespace_org_id FROM cm_monitors WHERE id = %s`
	return sqlf.Sprintf(
		ownerForId32Query,
		monitorID,
	), nil
}

//
// MonitorConnection
//
type monitorConnection struct {
	*Resolver
	monitors    []graphqlbackend.MonitorResolver
	totalCount  int32
	hasNextPage bool
}

func (m *monitorConnection) Nodes(ctx context.Context) ([]graphqlbackend.MonitorResolver, error) {
	return m.monitors, nil
}

func (m *monitorConnection) TotalCount(ctx context.Context) (int32, error) {
	return m.totalCount, nil
}

func (m *monitorConnection) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	if len(m.monitors) == 0 || !m.hasNextPage {
		return graphqlutil.HasNextPage(false), nil
	}
	return graphqlutil.NextPageCursor(string(m.monitors[len(m.monitors)-1].ID())), nil
}

//
// Monitor
//
type monitor struct {
	*Resolver
	*cm.Monitor
}

const (
	MonitorKind                     = "CodeMonitor"
	monitorTriggerQueryKind         = "CodeMonitorTriggerQuery"
	monitorTriggerEventKind         = "CodeMonitorTriggerEvent"
	monitorActionEmailKind          = "CodeMonitorActionEmail"
	monitorActionEventKind          = "CodeMonitorActionEmailEvent"
	monitorActionEmailRecipientKind = "CodeMonitorActionEmailRecipient"
)

func (m *monitor) ID() graphql.ID {
	return relay.MarshalID(MonitorKind, m.Monitor.ID)
}

func (m *monitor) CreatedBy(ctx context.Context) (*graphqlbackend.UserResolver, error) {
	return graphqlbackend.UserByIDInt32(ctx, m.store.Handle().DB(), m.Monitor.CreatedBy)
}

func (m *monitor) CreatedAt() graphqlbackend.DateTime {
	return graphqlbackend.DateTime{Time: m.Monitor.CreatedAt}
}

func (m *monitor) Description() string {
	return m.Monitor.Description
}

func (m *monitor) Enabled() bool {
	return m.Monitor.Enabled
}

func (m *monitor) Owner(ctx context.Context) (n graphqlbackend.NamespaceResolver, err error) {
	if m.NamespaceOrgID == nil {
		n.Namespace, err = graphqlbackend.UserByIDInt32(ctx, m.store.Handle().DB(), *m.NamespaceUserID)
	} else {
		n.Namespace, err = graphqlbackend.OrgByIDInt32(ctx, m.store.Handle().DB(), *m.NamespaceOrgID)
	}
	return n, err
}

func (m *monitor) Trigger(ctx context.Context) (graphqlbackend.MonitorTrigger, error) {
	t, err := m.store.TriggerQueryByMonitorIDInt64(ctx, m.Monitor.ID)
	if err != nil {
		return nil, err
	}
	return &monitorTrigger{&monitorQuery{m.Resolver, t}}, nil
}

func (m *monitor) Actions(ctx context.Context, args *graphqlbackend.ListActionArgs) (graphqlbackend.MonitorActionConnectionResolver, error) {
	return m.actionConnectionResolverWithTriggerID(ctx, nil, m.Monitor.ID, args)
}

func (r *Resolver) actionConnectionResolverWithTriggerID(ctx context.Context, triggerEventID *int, monitorID int64, args *graphqlbackend.ListActionArgs) (graphqlbackend.MonitorActionConnectionResolver, error) {
	// For now, we only support emails as actions. Once we add other actions such as
	// webhooks, we have to query those tables here too.
	q, err := r.store.ReadActionEmailQuery(ctx, monitorID, args)
	if err != nil {
		return nil, err
	}
	rows, err := r.store.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	es, err := cm.ScanEmails(rows)
	if err != nil {
		return nil, err
	}
	totalCount, err := r.store.TotalCountActionEmails(ctx, monitorID)
	if err != nil {
		return nil, err
	}
	actions := make([]graphqlbackend.MonitorAction, 0, len(es))
	for _, e := range es {
		actions = append(actions, &action{
			email: &monitorEmail{
				Resolver:       r,
				MonitorEmail:   e,
				triggerEventID: triggerEventID,
			},
		})
	}
	return &monitorActionConnection{actions: actions, totalCount: totalCount}, nil
}

//
// MonitorTrigger <<UNION>>
//
type monitorTrigger struct {
	query graphqlbackend.MonitorQueryResolver
}

func (t *monitorTrigger) ToMonitorQuery() (graphqlbackend.MonitorQueryResolver, bool) {
	return t.query, t.query != nil
}

//
// Query
//
type monitorQuery struct {
	*Resolver
	*cm.MonitorQuery
}

func (q *monitorQuery) ID() graphql.ID {
	return relay.MarshalID(monitorTriggerQueryKind, q.Id)
}

func (q *monitorQuery) Query() string {
	return q.QueryString
}

func (q *monitorQuery) Events(ctx context.Context, args *graphqlbackend.ListEventsArgs) (graphqlbackend.MonitorTriggerEventConnectionResolver, error) {
	es, err := q.store.GetEventsForQueryIDInt64(ctx, q.Id, args)
	if err != nil {
		return nil, err
	}
	totalCount, err := q.store.TotalCountEventsForQueryIDInt64(ctx, q.Id)
	if err != nil {
		return nil, err
	}
	events := make([]graphqlbackend.MonitorTriggerEventResolver, 0, len(es))
	for _, e := range es {
		events = append(events, graphqlbackend.MonitorTriggerEventResolver(&monitorTriggerEvent{
			Resolver:    q.Resolver,
			monitorID:   q.Monitor,
			TriggerJobs: e,
		}))
	}
	return &monitorTriggerEventConnection{Resolver: q.Resolver, events: events, totalCount: totalCount}, nil
}

//
// MonitorTriggerEventConnection
//
type monitorTriggerEventConnection struct {
	*Resolver
	events     []graphqlbackend.MonitorTriggerEventResolver
	totalCount int32
}

func (a *monitorTriggerEventConnection) Nodes(ctx context.Context) ([]graphqlbackend.MonitorTriggerEventResolver, error) {
	return a.events, nil
}

func (a *monitorTriggerEventConnection) TotalCount(ctx context.Context) (int32, error) {
	return a.totalCount, nil
}

func (a *monitorTriggerEventConnection) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	if len(a.events) == 0 {
		return graphqlutil.HasNextPage(false), nil
	}
	return graphqlutil.NextPageCursor(string(a.events[len(a.events)-1].ID())), nil
}

//
// MonitorTriggerEvent
//
type monitorTriggerEvent struct {
	*Resolver
	*cm.TriggerJobs
	monitorID int64
}

func (m *monitorTriggerEvent) ID() graphql.ID {
	return relay.MarshalID(monitorTriggerEventKind, m.Id)
}

// stateToStatus maps the state of the dbworker job to the public GraphQL status of
// events.
var stateToStatus = map[string]string{
	"completed":  "SUCCESS",
	"queued":     "PENDING",
	"processing": "PENDING",
	"errored":    "ERROR",
	"failed":     "ERROR",
}

func (m *monitorTriggerEvent) Status() (string, error) {
	if v, ok := stateToStatus[m.State]; ok {
		return v, nil
	}
	return "", errors.Errorf("unknown status: %s", m.State)
}

func (m *monitorTriggerEvent) Message() *string {
	return m.FailureMessage
}

func (m *monitorTriggerEvent) Timestamp() (graphqlbackend.DateTime, error) {
	if m.FinishedAt == nil {
		return graphqlbackend.DateTime{Time: m.store.Now()}, nil
	}
	return graphqlbackend.DateTime{Time: *m.FinishedAt}, nil
}

func (m *monitorTriggerEvent) Actions(ctx context.Context, args *graphqlbackend.ListActionArgs) (graphqlbackend.MonitorActionConnectionResolver, error) {
	return m.actionConnectionResolverWithTriggerID(ctx, &m.Id, m.monitorID, args)
}

// ActionConnection
//
type monitorActionConnection struct {
	actions    []graphqlbackend.MonitorAction
	totalCount int32
}

func (a *monitorActionConnection) Nodes(ctx context.Context) ([]graphqlbackend.MonitorAction, error) {
	return a.actions, nil
}

func (a *monitorActionConnection) TotalCount(ctx context.Context) (int32, error) {
	return a.totalCount, nil
}

func (a *monitorActionConnection) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	if len(a.actions) == 0 {
		return graphqlutil.HasNextPage(false), nil
	}
	last := a.actions[len(a.actions)-1]
	if email, ok := last.ToMonitorEmail(); ok {
		return graphqlutil.NextPageCursor(string(email.ID())), nil
	}
	return nil, errors.Errorf("we only support email actions for now")
}

//
// Action <<UNION>>
//
type action struct {
	email graphqlbackend.MonitorEmailResolver
}

func (a *action) ToMonitorEmail() (graphqlbackend.MonitorEmailResolver, bool) {
	return a.email, a.email != nil
}

//
// Email
//
type monitorEmail struct {
	*Resolver
	*cm.MonitorEmail

	// If triggerEventID == nil, all events of this action will be returned.
	// Otherwise, only those events of this action which are related to the specified
	// trigger event will be returned.
	triggerEventID *int
}

func (m *monitorEmail) Recipients(ctx context.Context, args *graphqlbackend.ListRecipientsArgs) (c graphqlbackend.MonitorActionEmailRecipientsConnectionResolver, err error) {
	var ms []*cm.Recipient
	ms, err = m.store.RecipientsForEmailIDInt64(ctx, m.Id, args)
	if err != nil {
		return nil, err
	}
	var ns []graphqlbackend.NamespaceResolver
	for _, r := range ms {
		n := graphqlbackend.NamespaceResolver{}
		if r.NamespaceOrgID == nil {
			n.Namespace, err = graphqlbackend.UserByIDInt32(ctx, m.store.Handle().DB(), *r.NamespaceUserID)
		} else {
			n.Namespace, err = graphqlbackend.OrgByIDInt32(ctx, m.store.Handle().DB(), *r.NamespaceOrgID)
		}
		if err != nil {
			return nil, err
		}
		ns = append(ns, n)
	}

	// Since recipients can either be a user or an org it would be very tedious to
	// use the user-id or org-id of the last entry as a cursor for the next page. It
	// is easier to just use the id of the recipients table.
	var nextPageCursor string
	if len(ms) > 0 {
		nextPageCursor = string(relay.MarshalID(monitorActionEmailRecipientKind, ms[len(ms)-1].ID))
	}

	var total int32
	total, err = m.store.TotalCountRecipients(ctx, m.Id)
	if err != nil {
		return nil, err
	}
	return &monitorActionEmailRecipientsConnection{ns, nextPageCursor, total}, nil
}

func (m *monitorEmail) Enabled() bool {
	return m.MonitorEmail.Enabled
}

func (m *monitorEmail) Priority() string {
	return m.MonitorEmail.Priority
}

func (m *monitorEmail) Header() string {
	return m.MonitorEmail.Header
}

func (m *monitorEmail) ID() graphql.ID {
	return relay.MarshalID(monitorActionEmailKind, m.Id)
}

func (m *monitorEmail) Events(ctx context.Context, args *graphqlbackend.ListEventsArgs) (graphqlbackend.MonitorActionEventConnectionResolver, error) {
	ajs, err := m.store.ReadActionEmailEvents(ctx, m.Id, m.triggerEventID, args)
	if err != nil {
		return nil, err
	}
	totalCount, err := m.store.TotalActionEmailEvents(ctx, m.Id, m.triggerEventID)
	if err != nil {
		return nil, err
	}
	events := make([]graphqlbackend.MonitorActionEventResolver, len(ajs))
	for i, aj := range ajs {
		events[i] = &monitorActionEvent{Resolver: m.Resolver, ActionJob: aj}
	}
	return &monitorActionEventConnection{events: events, totalCount: totalCount}, nil
}

//
// MonitorActionEmailRecipientConnection
//
type monitorActionEmailRecipientsConnection struct {
	recipients     []graphqlbackend.NamespaceResolver
	nextPageCursor string
	totalCount     int32
}

func (a *monitorActionEmailRecipientsConnection) Nodes(ctx context.Context) ([]graphqlbackend.NamespaceResolver, error) {
	return a.recipients, nil
}

func (a *monitorActionEmailRecipientsConnection) TotalCount(ctx context.Context) (int32, error) {
	return a.totalCount, nil
}

func (a *monitorActionEmailRecipientsConnection) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	if len(a.recipients) == 0 {
		return graphqlutil.HasNextPage(false), nil
	}
	return graphqlutil.NextPageCursor(a.nextPageCursor), nil
}

//
// MonitorActionEventConnection
//
type monitorActionEventConnection struct {
	events     []graphqlbackend.MonitorActionEventResolver
	totalCount int32
}

func (a *monitorActionEventConnection) Nodes(ctx context.Context) ([]graphqlbackend.MonitorActionEventResolver, error) {
	return a.events, nil
}

func (a *monitorActionEventConnection) TotalCount(ctx context.Context) (int32, error) {
	return a.totalCount, nil
}

func (a *monitorActionEventConnection) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	if len(a.events) == 0 {
		return graphqlutil.HasNextPage(false), nil
	}
	return graphqlutil.NextPageCursor(string(a.events[len(a.events)-1].ID())), nil
}

//
// MonitorEvent
//
type monitorActionEvent struct {
	*Resolver
	*cm.ActionJob
}

func (m *monitorActionEvent) ID() graphql.ID {
	return relay.MarshalID(monitorActionEventKind, m.Id)
}

func (m *monitorActionEvent) Status() (string, error) {
	status, ok := stateToStatus[m.State]
	if !ok {
		return "", errors.Errorf("unknown state: %s", m.State)
	}
	return status, nil
}

func (m *monitorActionEvent) Message() *string {
	return m.FailureMessage
}

func (m *monitorActionEvent) Timestamp() graphqlbackend.DateTime {
	if m.FinishedAt == nil {
		return graphqlbackend.DateTime{Time: m.store.Now()}
	}
	return graphqlbackend.DateTime{Time: *m.FinishedAt}
}
