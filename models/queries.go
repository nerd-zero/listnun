package models

import (
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Queries contains all prepared SQL queries.
type Queries struct {
	GetDashboardCharts   *sqlx.Stmt `query:"get-dashboard-charts"`
	GetDashboardCounts   *sqlx.Stmt `query:"get-dashboard-counts"`
	GetDashboardOverview *sqlx.Stmt `query:"get-dashboard-overview"`

	InsertSubscriber                *sqlx.Stmt `query:"insert-subscriber"`
	UpsertSubscriber                *sqlx.Stmt `query:"upsert-subscriber"`
	UpsertBlocklistSubscriber       *sqlx.Stmt `query:"upsert-blocklist-subscriber"`
	GetSubscriber                   *sqlx.Stmt `query:"get-subscriber"`
	HasSubscriberLists              *sqlx.Stmt `query:"has-subscriber-list"`
	GetSubscribersByEmails          *sqlx.Stmt `query:"get-subscribers-by-emails"`
	GetSubscriberLists              *sqlx.Stmt `query:"get-subscriber-lists"`
	GetSubscriptions                *sqlx.Stmt `query:"get-subscriptions"`
	GetSubscriberListsLazy          *sqlx.Stmt `query:"get-subscriber-lists-lazy"`
	UpdateSubscriber                *sqlx.Stmt `query:"update-subscriber"`
	UpdateSubscriberWithLists       *sqlx.Stmt `query:"update-subscriber-with-lists"`
	BlocklistSubscribers            *sqlx.Stmt `query:"blocklist-subscribers"`
	AddSubscribersToLists           *sqlx.Stmt `query:"add-subscribers-to-lists"`
	DeleteSubscriptions             *sqlx.Stmt `query:"delete-subscriptions"`
	DeleteUnconfirmedSubscriptions  *sqlx.Stmt `query:"delete-unconfirmed-subscriptions"`
	ConfirmSubscriptionOptin        *sqlx.Stmt `query:"confirm-subscription-optin"`
	UnsubscribeSubscribersFromLists *sqlx.Stmt `query:"unsubscribe-subscribers-from-lists"`
	DeleteSubscribers               *sqlx.Stmt `query:"delete-subscribers"`
	DeleteBlocklistedSubscribers    *sqlx.Stmt `query:"delete-blocklisted-subscribers"`
	DeleteOrphanSubscribers         *sqlx.Stmt `query:"delete-orphan-subscribers"`
	UnsubscribeByCampaign           *sqlx.Stmt `query:"unsubscribe-by-campaign"`
	ResubscribeByCampaign           *sqlx.Stmt `query:"resubscribe-by-campaign"`
	ExportSubscriberData            *sqlx.Stmt `query:"export-subscriber-data"`
	GetSubscriberActivity           *sqlx.Stmt `query:"get-subscriber-activity"`
	GetSubscriberEngagement         *sqlx.Stmt `query:"get-subscriber-engagement"`

	SetSubscriberScrubStatus         *sqlx.Stmt `query:"set-subscriber-scrub-status"`
	SetSubscribersScrubStatusByEmail *sqlx.Stmt `query:"set-subscribers-scrub-status-by-email"`
	GetRiskySubscriberIDs            *sqlx.Stmt `query:"get-risky-subscriber-ids"`

	AddUsageCounters *sqlx.Stmt `query:"add-usage-counters"`

	InsertScrubValidationBatch         *sqlx.Stmt `query:"insert-scrub-validation-batch"`
	GetScrubValidationBatch            *sqlx.Stmt `query:"get-scrub-validation-batch"`
	GetScrubValidationBatchesByJob     *sqlx.Stmt `query:"get-scrub-validation-batches-by-job"`
	UpdateScrubValidationBatchProgress *sqlx.Stmt `query:"update-scrub-validation-batch-progress"`
	DeleteScrubValidationBatch         *sqlx.Stmt `query:"delete-scrub-validation-batch"`

	InsertScrubValidationJob         *sqlx.Stmt `query:"insert-scrub-validation-job"`
	GetScrubValidationJob            *sqlx.Stmt `query:"get-scrub-validation-job"`
	GetActiveScrubJobForList         *sqlx.Stmt `query:"get-active-scrub-job-for-list"`
	GetActiveScrubJobsForTenant      *sqlx.Stmt `query:"get-active-scrub-jobs-for-tenant"`
	UpdateScrubValidationJobProgress *sqlx.Stmt `query:"update-scrub-validation-job-progress"`
	DeleteScrubValidationJob         *sqlx.Stmt `query:"delete-scrub-validation-job"`

	GetListSubscriberEmails *sqlx.Stmt `query:"get-list-subscriber-emails"`
	GetListScrubStatus      *sqlx.Stmt `query:"get-list-scrub-status"`
	UpdateListScrubResult   *sqlx.Stmt `query:"update-list-scrub-result"`

	// Non-prepared arbitrary subscriber queries.
	QuerySubscribers                       string     `query:"query-subscribers"`
	QuerySubscribersCount                  string     `query:"query-subscribers-count"`
	QuerySubscribersCountAll               *sqlx.Stmt `query:"query-subscribers-count-all"`
	QuerySubscribersForExport              string     `query:"query-subscribers-for-export"`
	QuerySubscribersTpl                    string     `query:"query-subscribers-template"`
	DeleteSubscribersByQuery               string     `query:"delete-subscribers-by-query"`
	AddSubscribersToListsByQuery           string     `query:"add-subscribers-to-lists-by-query"`
	BlocklistSubscribersByQuery            string     `query:"blocklist-subscribers-by-query"`
	DeleteSubscriptionsByQuery             string     `query:"delete-subscriptions-by-query"`
	UnsubscribeSubscribersFromListsByQuery string     `query:"unsubscribe-subscribers-from-lists-by-query"`

	CreateList        *sqlx.Stmt `query:"create-list"`
	QueryLists        string     `query:"query-lists"`
	GetLists          *sqlx.Stmt `query:"get-lists"`
	GetListsByOptin   *sqlx.Stmt `query:"get-lists-by-optin"`
	GetListTypes      *sqlx.Stmt `query:"get-list-types"`
	GetListIDsByUUIDs *sqlx.Stmt `query:"get-list-ids-by-uuids"`
	UpdateList        *sqlx.Stmt `query:"update-list"`
	UpdateListsDate   *sqlx.Stmt `query:"update-lists-date"`
	DeleteLists       *sqlx.Stmt `query:"delete-lists"`

	GetListsGrowth         *sqlx.Stmt `query:"get-lists-growth"`
	GetListActivity        *sqlx.Stmt `query:"get-list-activity"`
	GetListRecentCampaigns *sqlx.Stmt `query:"get-list-recent-campaigns"`
	GetOptinPending        *sqlx.Stmt `query:"get-optin-pending"`
	CountOptinPending      *sqlx.Stmt `query:"count-optin-pending"`
	MarkOptinSent          *sqlx.Stmt `query:"mark-optin-sent"`
	GetListsSummary        *sqlx.Stmt `query:"get-lists-summary"`

	CreateCampaign        *sqlx.Stmt `query:"create-campaign"`
	QueryCampaigns        string     `query:"query-campaigns"`
	GetCampaign           *sqlx.Stmt `query:"get-campaign"`
	GetCampaignForPreview *sqlx.Stmt `query:"get-campaign-for-preview"`
	GetCampaignStats      *sqlx.Stmt `query:"get-campaign-stats"`
	GetCampaignStatus     *sqlx.Stmt `query:"get-campaign-status"`

	SetCampaignEditor       *sqlx.Stmt `query:"set-campaign-editor"`
	GetCampaignsOverview    *sqlx.Stmt `query:"get-campaigns-overview"`
	GetCampaignStatusCounts *sqlx.Stmt `query:"get-campaign-status-counts"`

	GetSegments   *sqlx.Stmt `query:"get-segments"`
	GetSegment    *sqlx.Stmt `query:"get-segment"`
	CreateSegment *sqlx.Stmt `query:"create-segment"`
	UpdateSegment *sqlx.Stmt `query:"update-segment"`
	DeleteSegment *sqlx.Stmt `query:"delete-segment"`

	CountExistingEmails *sqlx.Stmt `query:"count-existing-emails"`
	RecordImport        *sqlx.Stmt `query:"record-import"`
	GetRecentImports    *sqlx.Stmt `query:"get-recent-imports"`

	GetForms             *sqlx.Stmt `query:"get-forms"`
	GetForm              *sqlx.Stmt `query:"get-form"`
	GetFormBySlug        *sqlx.Stmt `query:"get-form-by-slug"`
	FormSlugExists       *sqlx.Stmt `query:"form-slug-exists"`
	CreateForm           *sqlx.Stmt `query:"create-form"`
	UpdateForm           *sqlx.Stmt `query:"update-form"`
	DeleteForm           *sqlx.Stmt `query:"delete-form"`
	AddFormEvent         *sqlx.Stmt `query:"add-form-event"`
	GetFormStats         *sqlx.Stmt `query:"get-form-stats"`
	GetArchivedCampaigns *sqlx.Stmt `query:"get-archived-campaigns"`
	CampaignHasLists     *sqlx.Stmt `query:"campaign-has-lists"`

	// These two queries are read as strings and based on settings.individual_tracking=on/off,
	// are interpolated and copied to view and click counts. Same query, different tables.
	GetCampaignAnalyticsCounts  string     `query:"get-campaign-analytics-counts"`
	GetCampaignViewCounts       *sqlx.Stmt `query:"get-campaign-view-counts"`
	GetCampaignClickCounts      *sqlx.Stmt `query:"get-campaign-click-counts"`
	GetCampaignLinkCounts       *sqlx.Stmt `query:"get-campaign-link-counts"`
	GetCampaignBounceCounts     *sqlx.Stmt `query:"get-campaign-bounce-counts"`
	GetCampaignBounceTypeCounts *sqlx.Stmt `query:"get-campaign-bounce-type-counts"`
	DeleteCampaignViews         *sqlx.Stmt `query:"delete-campaign-views"`
	DeleteCampaignLinkClicks    *sqlx.Stmt `query:"delete-campaign-link-clicks"`
	ExportCampaignViews         *sqlx.Stmt `query:"export-campaign-views"`
	ExportCampaignLinkClicks    *sqlx.Stmt `query:"export-campaign-link-clicks"`

	NextCampaigns             *sqlx.Stmt `query:"next-campaigns"`
	GetRunningCampaign        *sqlx.Stmt `query:"get-running-campaign"`
	NextCampaignSubscribers   *sqlx.Stmt `query:"next-campaign-subscribers"`
	GetOneCampaignSubscriber  *sqlx.Stmt `query:"get-one-campaign-subscriber"`
	UpdateCampaign            *sqlx.Stmt `query:"update-campaign"`
	UpdateCampaignStatus      *sqlx.Stmt `query:"update-campaign-status"`
	SetCampaignPauseReason    *sqlx.Stmt `query:"set-campaign-pause-reason"`
	ClearCampaignPauseReason  *sqlx.Stmt `query:"clear-campaign-pause-reason"`
	GetRunningCampaignsByList *sqlx.Stmt `query:"get-running-campaigns-by-list"`
	GetAutoPausedCampaigns    *sqlx.Stmt `query:"get-auto-paused-campaigns"`
	UpdateCampaignCounts      *sqlx.Stmt `query:"update-campaign-counts"`
	UpdateCampaignArchive     *sqlx.Stmt `query:"update-campaign-archive"`
	RegisterCampaignView      *sqlx.Stmt `query:"register-campaign-view"`
	DeleteCampaign            *sqlx.Stmt `query:"delete-campaign"`
	DeleteCampaigns           *sqlx.Stmt `query:"delete-campaigns"`

	InsertMedia *sqlx.Stmt `query:"insert-media"`

	InsertActivity *sqlx.Stmt `query:"insert-activity"`

	CountInactiveSubscribers *sqlx.Stmt `query:"count-inactive-subscribers"`

	GetInactiveSubscriberIDs *sqlx.Stmt `query:"get-inactive-subscriber-ids"`

	UnsubscribeInactive *sqlx.Stmt `query:"unsubscribe-inactive"`

	InsertCleanupRun *sqlx.Stmt `query:"insert-cleanup-run"`

	FinishCleanupRun *sqlx.Stmt `query:"finish-cleanup-run"`

	GetLastCleanupRun *sqlx.Stmt `query:"get-last-cleanup-run"`

	GetActivity *sqlx.Stmt `query:"get-activity"`

	PruneActivity *sqlx.Stmt `query:"prune-activity"`

	GetMediaUsage *sqlx.Stmt `query:"get-media-usage"`

	GetUnsizedMedia *sqlx.Stmt `query:"get-unsized-media"`

	SetMediaSize *sqlx.Stmt `query:"set-media-size"`
	GetMedia     *sqlx.Stmt `query:"get-media"`
	QueryMedia   *sqlx.Stmt `query:"query-media"`
	DeleteMedia  *sqlx.Stmt `query:"delete-media"`

	GetWebhookEndpoints         *sqlx.Stmt `query:"get-webhook-endpoints"`
	GetWebhookEndpointSecret    *sqlx.Stmt `query:"get-webhook-endpoint-secret"`
	CreateWebhookEndpoint       *sqlx.Stmt `query:"create-webhook-endpoint"`
	UpdateWebhookEndpoint       *sqlx.Stmt `query:"update-webhook-endpoint"`
	RotateWebhookEndpointSecret *sqlx.Stmt `query:"rotate-webhook-endpoint-secret"`
	DeleteWebhookEndpoint       *sqlx.Stmt `query:"delete-webhook-endpoint"`
	EnqueueWebhookDeliveries    *sqlx.Stmt `query:"enqueue-webhook-deliveries"`
	EnqueueWebhookTest          *sqlx.Stmt `query:"enqueue-webhook-test"`
	ClaimWebhookDeliveries      *sqlx.Stmt `query:"claim-webhook-deliveries"`
	RecordWebhookAttempt        *sqlx.Stmt `query:"record-webhook-attempt"`
	QueryWebhookDeliveries      *sqlx.Stmt `query:"query-webhook-deliveries"`
	GetWebhookDelivery          *sqlx.Stmt `query:"get-webhook-delivery"`
	GetWebhookAttempts          *sqlx.Stmt `query:"get-webhook-attempts"`
	RetryWebhookDelivery        *sqlx.Stmt `query:"retry-webhook-delivery"`
	PruneWebhookDeliveries      *sqlx.Stmt `query:"prune-webhook-deliveries"`

	GetTenantReplies           *sqlx.Stmt `query:"get-tenant-replies"`
	GetReplyCampaign           *sqlx.Stmt `query:"get-reply-campaign"`
	InsertCampaignReply        *sqlx.Stmt `query:"insert-campaign-reply"`
	MarkCampaignReplyForwarded *sqlx.Stmt `query:"mark-campaign-reply-forwarded"`
	GetCampaignReplies         *sqlx.Stmt `query:"get-campaign-replies"`
	GetCampaignReplyCounts     *sqlx.Stmt `query:"get-campaign-reply-counts"`

	CreateTxMessage         *sqlx.Stmt `query:"create-tx-message"`
	SetTxMessageStatus      *sqlx.Stmt `query:"set-tx-message-status"`
	GetTxMessageForBounce   *sqlx.Stmt `query:"get-tx-message-for-bounce"`
	QueryTxMessages         *sqlx.Stmt `query:"query-tx-messages"`
	CountTxMessagesByStatus *sqlx.Stmt `query:"count-tx-messages-by-status"`
	GetTxMessage            *sqlx.Stmt `query:"get-tx-message"`
	GetTxMessageEvents      *sqlx.Stmt `query:"get-tx-message-events"`
	GetTxStats              *sqlx.Stmt `query:"get-tx-stats"`
	PruneTxMessages         *sqlx.Stmt `query:"prune-tx-messages"`

	GetAPIKeys         *sqlx.Stmt `query:"get-api-keys"`
	GetAPIKey          *sqlx.Stmt `query:"get-api-key"`
	GetAPIKeyAuth      *sqlx.Stmt `query:"get-api-key-auth"`
	CreateAPIKey       *sqlx.Stmt `query:"create-api-key"`
	UpdateAPIKeyScopes *sqlx.Stmt `query:"update-api-key-scopes"`
	RotateAPIKey       *sqlx.Stmt `query:"rotate-api-key"`
	RevokeAPIKey       *sqlx.Stmt `query:"revoke-api-key"`
	DeleteAPIKey       *sqlx.Stmt `query:"delete-api-key"`
	TouchAPIKey        *sqlx.Stmt `query:"touch-api-key"`

	CreateTemplate         *sqlx.Stmt `query:"create-template"`
	UpdateTemplateTxExtras *sqlx.Stmt `query:"update-template-tx-extras"`
	GetTxTemplateUsage     *sqlx.Stmt `query:"get-tx-template-usage"`
	GetTemplates           *sqlx.Stmt `query:"get-templates"`
	UpdateTemplate         *sqlx.Stmt `query:"update-template"`
	SetDefaultTemplate     *sqlx.Stmt `query:"set-default-template"`
	DeleteTemplate         *sqlx.Stmt `query:"delete-template"`

	CreateLink        *sqlx.Stmt `query:"create-link"`
	GetLinkURL        *sqlx.Stmt `query:"get-link-url"`
	RegisterLinkClick *sqlx.Stmt `query:"register-link-click"`

	GetSettings         *sqlx.Stmt `query:"get-settings"`
	UpdateSettings      *sqlx.Stmt `query:"update-settings"`
	UpdateSettingsByKey *sqlx.Stmt `query:"update-settings-by-key"`
	GetTenantRootURL    *sqlx.Stmt `query:"get-tenant-root-url"`

	// GetStats *sqlx.Stmt `query:"get-stats"`
	RecordBounce                *sqlx.Stmt `query:"record-bounce"`
	QueryBounces                string     `query:"query-bounces"`
	GetBounceTypeCounts         string     `query:"get-bounce-type-counts"`
	BlocklistBouncedSubscribers *sqlx.Stmt `query:"blocklist-bounced-subscribers"`
	DeleteBounces               *sqlx.Stmt `query:"delete-bounces"`
	DeleteBouncesBySubscriber   *sqlx.Stmt `query:"delete-bounces-by-subscriber"`
	GetDBInfo                   string     `query:"get-db-info"`

	CreateUser         *sqlx.Stmt `query:"create-user"`
	UpdateUser         *sqlx.Stmt `query:"update-user"`
	UpdateUserProfile  *sqlx.Stmt `query:"update-user-profile"`
	UpdateUserLogin    *sqlx.Stmt `query:"update-user-login"`
	SetUserTwoFA       *sqlx.Stmt `query:"set-user-twofa"`
	DeleteUsers        *sqlx.Stmt `query:"delete-users"`
	HasUsers           *sqlx.Stmt `query:"has-users"`
	GetUsers           *sqlx.Stmt `query:"get-users"`
	GetUser            *sqlx.Stmt `query:"get-user"`
	GetAPITokens       *sqlx.Stmt `query:"get-api-tokens"`
	LoginUser          *sqlx.Stmt `query:"login-user"`
	DeleteUserSessions *sqlx.Stmt `query:"delete-user-sessions"`

	CreateRole            *sqlx.Stmt `query:"create-role"`
	GetUserRoles          *sqlx.Stmt `query:"get-user-roles"`
	GetListRoles          *sqlx.Stmt `query:"get-list-roles"`
	UpdateRole            *sqlx.Stmt `query:"update-role"`
	DeleteRole            *sqlx.Stmt `query:"delete-role"`
	UpsertListPermissions *sqlx.Stmt `query:"upsert-list-permissions"`
	DeleteListPermission  *sqlx.Stmt `query:"delete-list-permission"`

	GetTenantBySlug           *sqlx.Stmt `query:"get-tenant-by-slug"`
	GetTenantByCustomDomain   *sqlx.Stmt `query:"get-tenant-by-custom-domain"`
	GetActiveTenantIDs        *sqlx.Stmt `query:"get-active-tenant-ids"`
	GetTenantOrganizationName *sqlx.Stmt `query:"get-tenant-organization-name"`
}

// compileSubscriberQueryTpl takes an arbitrary WHERE expressions
// to filter subscribers from the subscribers table and prepares a query
// out of it using the raw `query-subscribers-template` query template.
// The dry run ($1=true) makes query-subscribers-template's body resolve
// to `LIMIT (CASE WHEN $1 THEN 1 END)` - a plain SELECT, incapable of
// mutating anything regardless of $1 - so this is safe to run directly on
// whatever Execer the caller passes (a *sqlx.Tx opened by Core.WithTenant,
// or a *sqlx.DB for callers not yet tenant-scoped) without needing its own
// nested transaction (which wouldn't be valid inside an already-open one
// anyway - *sqlx.Tx has no BeginTxx of its own).
func (q *Queries) compileSubscriberQueryTpl(searchStr, queryExp string, db sqlx.Execer, subStatus string) (string, error) {
	// There's an arbitrary query condition.
	cond := "TRUE"
	if queryExp != "" {
		cond = queryExp
	}

	// Perform the dry run.
	stmt := strings.ReplaceAll(q.QuerySubscribersTpl, "%query%", cond)
	if _, err := db.Exec(stmt, true, pq.Int64Array{}, subStatus, searchStr); err != nil {
		return "", err
	}

	return stmt, nil
}

// compileSubscriberQueryTpl takes an arbitrary WHERE expressions and a subscriber
// query template that depends on the filter (eg: delete by query, blocklist by query etc.)
// combines and executes them. db is typically a *sqlx.Tx opened by the
// caller via Core.WithTenant so the mutation runs under the right tenant
// context; existing callers not yet tenant-scoped keep passing a plain
// *sqlx.DB unchanged.
func (q *Queries) ExecSubQueryTpl(searchStr, queryExp, baseQueryTpl string, listIDs []int, db sqlx.Execer, subStatus string, args ...any) error {
	// Perform a dry run.
	filterExp, err := q.compileSubscriberQueryTpl(searchStr, queryExp, db, subStatus)
	if err != nil {
		return err
	}

	if len(listIDs) == 0 {
		listIDs = []int{}
	}

	// Insert the subscriber filter query into the target query.
	stmt := strings.ReplaceAll(baseQueryTpl, "%query%", filterExp)

	// First argument is the boolean indicating if the query is a dry run.
	a := append([]any{false, pq.Array(listIDs), subStatus, searchStr}, args...)

	// Execute the query on the DB.
	if _, err := db.Exec(stmt, a...); err != nil {
		return err
	}
	return nil
}
