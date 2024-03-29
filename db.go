package main

import (
	"database/sql"
	_ "github.com/lib/pq"
)

type Account struct {
	Id                int
	UserId            string
	MinecraftUuid     string
	MinecraftUsername string
	IsMain            bool
}

type Db struct {
	db *sql.DB
}

func Connect(url string) (*Db, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	var re Db
	re.db = db

	return &re, nil
}

func (r *Db) Close() {
	r.Close()
}

func (r *Db) AddAccount(account Account) error {

	_, err := r.db.Query(
		"INSERT INTO accounts (user_id, minecraft_uuid, minecraft_username, is_main)  VALUES ($1, $2, $3, $4)",
		account.UserId,
		account.MinecraftUuid,
		account.MinecraftUsername,
		account.IsMain,
	)

	return err
}

func (r *Db) DeleteAccount(id int) error {

	_, err := r.db.Query(
		"DELETE FROM accounts WHERE id = $1",
		id,
	)

	return err
}

func (r *Db) GetAccountByUsername(username string) (*Account, error) {

	var re Account

	err := r.db.QueryRow(
		"SELECT id, user_id, minecraft_uuid, minecraft_username, is_main FROM accounts WHERE minecraft_username = $1",
		username,
	).Scan(
		&re.Id,
		&re.UserId,
		&re.MinecraftUuid,
		&re.MinecraftUsername,
		&re.IsMain,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &re, nil
}

func (r *Db) GetAccountsByUser(id string) ([]Account, error) {

	var re []Account

	rows, err := r.db.Query(
		"SELECT id, user_id, minecraft_uuid, minecraft_username, is_main FROM accounts WHERE user_id = $1",
		id,
	)
	if err != nil {
		return re, err
	}

	for rows.Next() {
		var a Account

		err = rows.Scan(
			&a.Id,
			&a.UserId,
			&a.MinecraftUuid,
			&a.MinecraftUsername,
			&a.IsMain,
		)
		if err != nil {
			return re, err
		}

		re = append(re, a)
	}
	if err = rows.Err(); err != nil {
		return re, err
	}

	return re, err
}
