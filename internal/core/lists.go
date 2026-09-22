package core

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

type listType struct {
	ID   int    `json:"id"`
	UUID string `json:"uuid"`
	Type string `json:"type"`
}

// GetLists gets all lists optionally filtered by type and status.
func (c *Core) GetLists(ctx context.Context, tenantID int, typ, status string, getAll bool, permittedIDs []int) ([]models.List, error) {
	out := []models.List{}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetLists).Select(&out, typ, status, "id", getAll, pq.Array(permittedIDs))
	})
	if err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	// Replace null tags.
	for i, l := range out {
		if l.Tags == nil {
			out[i].Tags = []string{}
		}

		// Total counts.
		for _, c := range l.SubscriberCounts {
			out[i].SubscriberCount += c
		}
	}

	return out, nil
}

// QueryLists gets multiple lists based on multiple query params. Along with the  paginated and sliced
// results, the total number of lists in the DB is returned.
func (c *Core) QueryLists(ctx context.Context, tenantID int, searchStr, typ, optin, status string, tags []string, orderBy, order string, getAll bool, permittedIDs []int, offset, limit int) ([]models.List, int, error) {
	_ = c.refreshCache(matListSubStats, false)

	if tags == nil {
		tags = []string{}
	}

	var (
		out            = []models.List{}
		queryStr, stmt = makeSearchQuery(searchStr, orderBy, order, c.q.QueryLists, listQuerySortFields)
	)
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return tx.Select(&out, stmt, 0, "", queryStr, typ, optin, status, pq.StringArray(tags), getAll, pq.Array(permittedIDs), offset, limit)
	})
	if err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	total := 0
	if len(out) > 0 {
		total = out[0].Total

		// Replace null tags.
		for i, l := range out {
			if l.Tags == nil {
				out[i].Tags = []string{}
			}
		}
	}

	return out, total, nil
}

// GetList gets a list by its ID or UUID.
func (c *Core) GetList(ctx context.Context, tenantID int, id int, uuid string) (models.List, error) {
	var uu any
	if uuid != "" {
		uu = uuid
	}

	var res []models.List
	queryStr, stmt := makeSearchQuery("", "", "", c.q.QueryLists, nil)
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return tx.Select(&res, stmt, id, uu, queryStr, "", "", "", pq.StringArray{}, true, nil, 0, 1)
	})
	if err != nil {
		c.log.Printf("error fetching lists: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	if len(res) == 0 {
		return models.List{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	out := res[0]
	if out.Tags == nil {
		out.Tags = []string{}
	}
	// Total counts.
	for _, c := range out.SubscriberCounts {
		out.SubscriberCount += c
	}

	return out, nil
}

// GetListsByOptin returns lists by optin type.
func (c *Core) GetListsByOptin(ctx context.Context, tenantID int, ids []int, optinType string) ([]models.List, error) {
	out := []models.List{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetListsByOptin).Select(&out, optinType, pq.Array(ids), nil)
	})
	if err != nil {
		c.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	return out, nil
}

// GetListTypes returns lists by their IDs or UUIDs.
// If ids is given, then the map returned has the list IDs as keys,
// otherwise, they have UUIDs as the keys.
// Note: This is a really weird and awkward API. Ideally, Go Generics
// should've somehow supported generic struct methods.
func (c *Core) GetListTypes(ctx context.Context, tenantID int, ids []int, uuids []string) (map[any]string, error) {
	res := []listType{}

	out := map[any]string{}
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetListTypes).Select(&res, pq.Array(ids), pq.StringArray(uuids))
	})
	if err != nil {
		c.log.Printf("error fetching list types: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	isIDs := ids != nil
	for _, r := range res {
		if isIDs {
			out[r.ID] = r.Type
		} else {
			out[r.UUID] = r.Type
		}
	}

	return out, nil
}

// GetListIDsByUUIDs resolves list UUIDs to IDs, tenant-scoped -- used by
// processSubForm (public.go), which works in UUIDs throughout, to get the
// int list IDs the Scrub risky-subscriber auto-pause path needs.
func (c *Core) GetListIDsByUUIDs(ctx context.Context, tenantID int, uuids []string) ([]int, error) {
	var ids []int
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.GetListIDsByUUIDs).Select(&ids, tenantID, pq.StringArray(uuids))
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}
	return ids, nil
}

// CreateList creates a new list.
func (c *Core) CreateList(ctx context.Context, tenantID int, l models.List) (models.List, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		c.log.Printf("error generating UUID: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	if l.Type == "" {
		l.Type = models.ListTypePrivate
	}
	if l.Optin == "" {
		l.Optin = models.ListOptinSingle
	}
	if l.Status == "" {
		l.Status = models.ListStatusActive
	}

	// Insert and read ID.
	var newID int
	l.UUID = uu.String()
	err = c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		return stmtx(tx, c.q.CreateList).Get(&newID, l.UUID, l.Name, l.Type, l.Optin, l.Status, pq.StringArray(normalizeTags(l.Tags)), l.Description, tenantID)
	})
	if err != nil {
		c.log.Printf("error creating list: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	return c.GetList(ctx, tenantID, newID, "")
}

// UpdateList updates a given list.
func (c *Core) UpdateList(ctx context.Context, tenantID int, id int, l models.List) (models.List, error) {
	var n int64
	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		res, err := stmtx(tx, c.q.UpdateList).Exec(id, l.Name, l.Type, l.Optin, l.Status, pq.StringArray(normalizeTags(l.Tags)), l.Description)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		return err
	})
	if err != nil {
		c.log.Printf("error updating list: %v", err)
		return models.List{}, echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	if n == 0 {
		return models.List{}, echo.NewHTTPError(http.StatusBadRequest,
			c.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.list}"))
	}

	return c.GetList(ctx, tenantID, id, "")
}

// DeleteList deletes a list.
func (c *Core) DeleteList(ctx context.Context, tenantID int, id int) error {
	return c.DeleteLists(ctx, tenantID, []int{id}, "", true, nil)
}

// DeleteLists deletes multiple lists.
func (c *Core) DeleteLists(ctx context.Context, tenantID int, ids []int, query string, getAll bool, permittedIDs []int) error {
	var queryStr string

	if len(ids) > 0 {
		queryStr = ""
	} else {
		queryStr = makeSearchString(query)
	}

	err := c.WithTenant(ctx, tenantID, nil, func(tx *sqlx.Tx) error {
		_, err := stmtx(tx, c.q.DeleteLists).Exec(pq.Array(ids), queryStr, getAll, pq.Array(permittedIDs))
		return err
	})
	if err != nil {
		c.log.Printf("error deleting lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			c.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}
	return nil
}
