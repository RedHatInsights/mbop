package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	l "github.com/redhatinsights/mbop/internal/logger"
)

type postgresStore struct {
	db *sql.DB
}

func (p *postgresStore) All() ([]Registration, error) {
	rows, err := p.db.Query(`select id, org_id, uid, extra from registrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Registration, 0)
	for rows.Next() {
		r, err := scanRegistration(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}

	return out, nil
}

func (p *postgresStore) Find(orgID, uid string) (*Registration, error) {
	rows := p.db.QueryRow(
		`select id, org_id, uid, extra from registrations where org_id = $1 or uid = $2 limit 1`,
		orgID,
		uid,
	)
	return scanRegistration(rows)
}

func (p *postgresStore) Create(r *Registration) (string, error) {
	res := p.db.QueryRow(
		`insert into registrations (org_id, uid, extra) values ($1, $2, $3) returning id`,
		r.OrgID,
		r.UID,
		r.Extra,
	)

	var id string
	err := res.Scan(&id)
	if err != nil {
		return "", err
	}

	l.Log.Info("Created registration", "id", id, "org_id", r.OrgID, "uid", r.UID)
	return id, nil
}

func (p *postgresStore) Update(r *Registration, update *RegistrationUpdate) error {
	//TODO: maybe more fields someday, not sure.
	_, err := p.db.Exec(
		`update registrations set extra = $1 where org_id = $2 and uid = $3`,
		update.Extra,
		r.OrgID,
		r.UID,
	)

	return err
}

func (p *postgresStore) Delete(orgID, uid string) error {
	res, err := p.db.Exec(
		`delete from registrations where org_id = $1 and uid = $2`,
		orgID,
		uid,
	)
	if err != nil {
		return err
	}

	count, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if count != 1 {
		return fmt.Errorf("no registration found for org_id: %v, uid: %v", orgID, uid)
	}

	l.Log.Info("Deleted registration", "orgID", orgID, "uid", uid)
	return nil
}

// implement our own teeny scanner interface so we can use both sql.Row and/or sql.Rows
type scanner interface {
	Scan(dest ...any) error
}

func scanRegistration(row scanner) (*Registration, error) {
	var (
		id    string
		orgID string
		uid   string
		extra []byte
	)
	err := row.Scan(&id, &orgID, &uid, &extra)
	if err != nil {
		return nil, err
	}

	var e map[string]any
	if extra != nil {
		err := json.Unmarshal(extra, &e)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal extra json")
		}
	}

	return &Registration{
		ID:    id,
		OrgID: orgID,
		UID:   uid,
		Extra: e,
	}, nil
}