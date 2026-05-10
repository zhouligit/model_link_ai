package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type Org struct {
	ID          uint64
	Name        string
	Slug        sql.NullString
	OwnerUserID uint64
	Status      string
}

func (s *Store) CreateOrg(ctx context.Context, name string, ownerID uint64, slug string) (uint64, error) {
	name = strings.TrimSpace(name)
	var slugPtr interface{}
	if strings.TrimSpace(slug) != "" {
		slugPtr = strings.TrimSpace(slug)
	}
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO orgs (name, slug, owner_user_id, status) VALUES (?, ?, ?, 'active')`,
		name, slugPtr, ownerID,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	oid := uint64(id)
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO org_members (org_id, user_id, role) VALUES (?, ?, 'owner')`,
		oid, ownerID,
	)
	return oid, err
}

func (s *Store) ListUserOrgs(ctx context.Context, userID uint64) ([]Org, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT o.id, o.name, o.slug, o.owner_user_id, o.status
		 FROM orgs o INNER JOIN org_members m ON m.org_id = o.id
		 WHERE m.user_id = ? AND o.status = 'active' ORDER BY o.id`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Org
	for rows.Next() {
		var o Org
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.OwnerUserID, &o.Status); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) GetOrg(ctx context.Context, orgID uint64) (*Org, error) {
	var o Org
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, name, slug, owner_user_id, status FROM orgs WHERE id = ?`,
		orgID,
	).Scan(&o.ID, &o.Name, &o.Slug, &o.OwnerUserID, &o.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &o, err
}

func (s *Store) IsOrgMember(ctx context.Context, orgID, userID uint64) (role string, ok bool, err error) {
	err = s.DB.QueryRowContext(ctx,
		`SELECT role FROM org_members WHERE org_id = ? AND user_id = ?`,
		orgID, userID,
	).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return role, true, nil
}
