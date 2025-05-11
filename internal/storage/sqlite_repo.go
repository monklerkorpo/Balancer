package storage

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

type SQLiteClientRepo struct {
	db *sql.DB
}

func NewSQLiteClientRepo(dbPath string) (*SQLiteClientRepo, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Создаем таблицу, если ее нет
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS clients (
		client_id TEXT PRIMARY KEY,
		capacity INTEGER,
		refill_rate INTEGER
	)`)
	if err != nil {
		return nil, err
	}

	return &SQLiteClientRepo{db: db}, nil
}

func (r *SQLiteClientRepo) Create(l ClientLimit) error {
	query := `INSERT INTO clients (client_id, capacity, refill_rate) VALUES (?, ?, ?)`
	_, err := r.db.Exec(query, l.ClientID, l.Capacity, l.RefillRate)
	return err
}

func (r *SQLiteClientRepo) Get(id string) (ClientLimit, error) {
	var l ClientLimit
	query := `SELECT client_id, capacity, refill_rate FROM clients WHERE client_id = ?`
	row := r.db.QueryRow(query, id)

	err := row.Scan(&l.ClientID, &l.Capacity, &l.RefillRate)
	if err != nil {
		if err == sql.ErrNoRows {
			return ClientLimit{}, ErrNotFound
		}
		return ClientLimit{}, err
	}

	return l, nil
}

func (r *SQLiteClientRepo) Update(l ClientLimit) error {
	query := `UPDATE clients SET capacity = ?, refill_rate = ? WHERE client_id = ?`
	result, err := r.db.Exec(query, l.Capacity, l.RefillRate, l.ClientID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *SQLiteClientRepo) Delete(id string) error {
	query := `DELETE FROM clients WHERE client_id = ?`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *SQLiteClientRepo) List() ([]ClientLimit, error) {
	var clients []ClientLimit
	query := `SELECT client_id, capacity, refill_rate FROM clients`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var l ClientLimit
		if err := rows.Scan(&l.ClientID, &l.Capacity, &l.RefillRate); err != nil {
			return nil, err
		}
		clients = append(clients, l)
	}

	return clients, nil
}
