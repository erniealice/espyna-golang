//go:build postgresql

package ledger

import (
	"context"
	"fmt"
	"time"

	journalentrypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/journal_entry"
)

// CountByStatus returns counts of journal entries grouped by status (DRAFT/POSTED/REVERSED)
// optionally filtered to those created at-or-after `since`.
// Workspace-scoped via the workspace_id column.
//
// Performance index recommendation:
//
//	CREATE INDEX idx_journal_entry_workspace_status_created
//	  ON journal_entry(workspace_id, status, date_created DESC) WHERE active = true;
func (r *PostgresJournalEntryRepository) CountByStatus(
	ctx context.Context,
	workspaceID string,
	since time.Time,
) (map[string]int64, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	var (
		query    string
		args     []any
	)

	if since.IsZero() {
		query = `
			SELECT je.status, COUNT(*)::bigint
			FROM journal_entry je
			WHERE je.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR je.workspace_id = $1)
			GROUP BY je.status`
		args = []any{workspaceID}
	} else {
		query = `
			SELECT je.status, COUNT(*)::bigint
			FROM journal_entry je
			WHERE je.active = true
			  AND je.date_created >= $2
			  AND ($1::text IS NULL OR $1::text = '' OR je.workspace_id = $1)
			GROUP BY je.status`
		args = []any{workspaceID, since}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return map[string]int64{}, nil //nolint:nilerr
	}
	defer rows.Close()

	out := make(map[string]int64, 3)
	for rows.Next() {
		var (
			status string
			n      int64
		)
		if scanErr := rows.Scan(&status, &n); scanErr != nil {
			return nil, fmt.Errorf("failed to scan journal_entry count row: %w", scanErr)
		}
		out[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating journal_entry count rows: %w", err)
	}
	return out, nil
}

// RecentEntries returns the most recent journal entries, ordered by date_created DESC.
// Workspace-scoped.
func (r *PostgresJournalEntryRepository) RecentEntries(
	ctx context.Context,
	workspaceID string,
	limit int32,
) ([]*journalentrypb.JournalEntry, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}
	if limit <= 0 {
		limit = 5
	}

	query := `
		SELECT
			je.id,
			je.entry_number,
			je.description,
			je.entry_date,
			je.status,
			je.source_type,
			je.total_debit,
			je.total_credit,
			je.date_created
		FROM journal_entry je
		WHERE je.active = true
		  AND ($1::text IS NULL OR $1::text = '' OR je.workspace_id = $1)
		ORDER BY je.date_created DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent journal entries: %w", err)
	}
	defer rows.Close()

	var out []*journalentrypb.JournalEntry
	for rows.Next() {
		var (
			id          string
			entryNumber string
			description string
			entryDate   *time.Time
			status      string
			sourceType  string
			totalDebit  int64
			totalCredit int64
			dateCreated time.Time
		)
		if scanErr := rows.Scan(
			&id,
			&entryNumber,
			&description,
			&entryDate,
			&status,
			&sourceType,
			&totalDebit,
			&totalCredit,
			&dateCreated,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan recent journal entry row: %w", scanErr)
		}

		entry := &journalentrypb.JournalEntry{
			Id:          id,
			EntryNumber: entryNumber,
			Description: description,
			TotalDebit:  totalDebit,
			TotalCredit: totalCredit,
			Active:      true,
			Status:      parseJournalEntryStatus(status),
			SourceType:  parseJournalSourceType(sourceType),
		}
		if entryDate != nil && !entryDate.IsZero() {
			entry.EntryDate = entryDate.UnixMilli()
			s := entryDate.Format("2006-01-02")
			entry.EntryDateString = &s
		}
		if !dateCreated.IsZero() {
			ms := dateCreated.UnixMilli()
			entry.DateCreated = &ms
			s := dateCreated.Format(time.RFC3339)
			entry.DateCreatedString = &s
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recent journal entry rows: %w", err)
	}
	return out, nil
}
