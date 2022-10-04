package sql

import (
	"context"
	"database/sql"
	"time"

	"github.com/zitadel/logging"

	"github.com/zitadel/zitadel/internal/errors"
	es_models "github.com/zitadel/zitadel/internal/eventstore/v1/models"
	"github.com/zitadel/zitadel/internal/telemetry/tracing"
)

type Querier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

func (db *SQL) Filter(ctx context.Context, searchQuery *es_models.SearchQueryFactory) (events []*es_models.Event, err error) {
	return filter(db.client, searchQuery)
}

func filter(querier Querier, searchQuery *es_models.SearchQueryFactory) (events []*es_models.Event, err error) {
	query, limit, values, rowScanner := buildQuery(searchQuery)
	if query == "" {
		return nil, errors.ThrowInvalidArgument(nil, "SQL-rWeBw", "invalid query factory")
	}

	rows, err := querier.Query(query, values...)
	if err != nil {
		logging.New().WithError(err).Info("query failed")
		return nil, errors.ThrowInternal(err, "SQL-IJuyR", "unable to filter events")
	}
	defer rows.Close()

	events = make([]*es_models.Event, 0, limit)

	for rows.Next() {
		event := new(es_models.Event)
		err := rowScanner(rows.Scan, event)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

func (db *SQL) LatestCreationDate(ctx context.Context, queryFactory *es_models.SearchQueryFactory) (creationDate time.Time, err error) {
	query, _, values, rowScanner := buildQuery(queryFactory)
	if query == "" {
		return time.Time{}, errors.ThrowInvalidArgument(nil, "SQL-rWeBw", "invalid query factory")
	}

	row := db.client.QueryRow(query, values...)
	if err = rowScanner(row.Scan, creationDate); err != nil {
		logging.New().WithError(err).WithField("traceID", tracing.TraceIDFromCtx(ctx)).Info("query failed")
		return time.Time{}, errors.ThrowInternal(err, "SQL-Yczyx", "unable to filter latest sequence")
	}
	return creationDate, nil
}

func (db *SQL) InstanceIDs(ctx context.Context, queryFactory *es_models.SearchQueryFactory) ([]string, error) {
	query, _, values, rowScanner := buildQuery(queryFactory)
	if query == "" {
		return nil, errors.ThrowInvalidArgument(nil, "SQL-Sfwg2", "invalid query factory")
	}

	rows, err := db.client.Query(query, values...)
	if err != nil {
		logging.New().WithError(err).Info("query failed")
		return nil, errors.ThrowInternal(err, "SQL-Sfg3r", "unable to filter instance ids")
	}
	defer rows.Close()

	ids := make([]string, 0)

	for rows.Next() {
		var id string
		err := rowScanner(rows.Scan, &id)
		if err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}
