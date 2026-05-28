package DAOS

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rstrom1763/plex_comparisons/constants"
	"github.com/rstrom1763/plex_comparisons/structs"
)

type LocalStateDAO struct {
	db *sql.DB
}

type localStateTable struct {
	name string
	sql  string
}

var localStateTables = []localStateTable{
	{name: "servers", sql: constants.CREATE_SERVERS_TABLE},
	{name: "saved filters", sql: constants.CREATE_SAVED_FILTERS_TABLE},
	{name: "trusted servers", sql: constants.CREATE_TRUSTED_SERVERS_TABLE},
	{name: "users", sql: constants.CREATE_USERS_TABLE},
}

func NewLocalStateDAO(dbPath string) (*LocalStateDAO, error) {
	return newLocalStateDAO(dbPath, "sqlite3")
}

func newLocalStateDAO(dbPath string, driverName string) (*LocalStateDAO, error) {
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open local state database: %w", err)
	}

	if err := ensureLocalStateTables(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &LocalStateDAO{db: db}, nil
}

func ensureLocalStateTables(db *sql.DB) error {
	var tableErrors []error
	for _, table := range localStateTables {
		if _, err := db.Exec(table.sql); err != nil {
			tableErrors = append(tableErrors, fmt.Errorf("could not ensure %s table exists: %w", table.name, err))
		}
	}

	return errors.Join(tableErrors...)
}

func (dao *LocalStateDAO) Close() error {
	return dao.db.Close()
}

func (dao *LocalStateDAO) AddServer(server structs.Server) error {
	_, err := dao.db.Exec(constants.INSERT_SERVER, server.Name, server.Address, server.Token)
	if err != nil {
		return fmt.Errorf("could not add server: %w", err)
	}
	return nil
}

func (dao *LocalStateDAO) GetServers() ([]structs.Server, error) {
	rows, err := dao.db.Query(constants.SELECT_ALL_SERVERS)
	if err != nil {
		return nil, fmt.Errorf("could not query servers: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var servers []structs.Server
	for rows.Next() {
		var s structs.Server
		if err := rows.Scan(&s.ID, &s.Name, &s.Address, &s.Token); err != nil {
			return nil, fmt.Errorf("could not scan server: %w", err)
		}
		servers = append(servers, s)
	}
	return servers, nil
}

func (dao *LocalStateDAO) DeleteServer(id int) error {
	_, err := dao.db.Exec(constants.DELETE_SERVER, id)
	if err != nil {
		return fmt.Errorf("could not delete server: %w", err)
	}
	return nil
}

func (dao *LocalStateDAO) UpdateServer(server structs.Server) error {
	_, err := dao.db.Exec(constants.UPDATE_SERVER, server.Name, server.Address, server.Token, server.ID)
	if err != nil {
		return fmt.Errorf("could not update server: %w", err)
	}
	return nil
}

func (dao *LocalStateDAO) AddSavedFilter(filter structs.SavedFilter) (int64, error) {
	result, err := dao.db.Exec(constants.INSERT_SAVED_FILTER, filter.Name, filter.FilterData)
	if err != nil {
		return 0, fmt.Errorf("could not add saved filter: %w", err)
	}
	return result.LastInsertId()
}

func (dao *LocalStateDAO) GetSavedFilters() ([]structs.SavedFilter, error) {
	rows, err := dao.db.Query(constants.SELECT_ALL_SAVED_FILTERS)
	if err != nil {
		return nil, fmt.Errorf("could not query saved filters: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var filters []structs.SavedFilter
	for rows.Next() {
		var f structs.SavedFilter
		if err := rows.Scan(&f.ID, &f.Name, &f.FilterData); err != nil {
			return nil, fmt.Errorf("could not scan saved filter: %w", err)
		}
		filters = append(filters, f)
	}
	return filters, nil
}

func (dao *LocalStateDAO) DeleteSavedFilter(id int) error {
	_, err := dao.db.Exec(constants.DELETE_SAVED_FILTER, id)
	if err != nil {
		return fmt.Errorf("could not delete saved filter: %w", err)
	}
	return nil
}

func (dao *LocalStateDAO) UpdateSavedFilter(filter structs.SavedFilter) error {
	_, err := dao.db.Exec(constants.UPDATE_SAVED_FILTER, filter.Name, filter.FilterData, filter.ID)
	if err != nil {
		return fmt.Errorf("could not update saved filter: %w", err)
	}
	return nil
}

func (dao *LocalStateDAO) AddTrustedServer(name string, tokenHash string) error {
	_, err := dao.db.Exec(constants.INSERT_TRUSTED_SERVER, name, tokenHash)
	if err != nil {
		return fmt.Errorf("could not add trusted server: %w", err)
	}
	return nil
}

func (dao *LocalStateDAO) GetTrustedServers() ([]structs.TrustedServer, error) {
	rows, err := dao.db.Query(constants.SELECT_ALL_TRUSTED_SERVERS)
	if err != nil {
		return nil, fmt.Errorf("could not query trusted servers: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var servers []structs.TrustedServer
	for rows.Next() {
		var s structs.TrustedServer
		if err := rows.Scan(&s.ID, &s.Name, &s.TokenHash); err != nil {
			return nil, fmt.Errorf("could not scan trusted server: %w", err)
		}
		servers = append(servers, s)
	}
	return servers, nil
}

func (dao *LocalStateDAO) GetTrustedServerByName(name string) (*structs.TrustedServer, error) {
	var s structs.TrustedServer
	err := dao.db.QueryRow(constants.SELECT_TRUSTED_SERVER_BY_NAME, name).Scan(&s.ID, &s.Name, &s.TokenHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get trusted server by name: %w", err)
	}
	return &s, nil
}

func (dao *LocalStateDAO) DeleteTrustedServer(id int) error {
	_, err := dao.db.Exec(constants.DELETE_TRUSTED_SERVER, id)
	if err != nil {
		return fmt.Errorf("could not delete trusted server: %w", err)
	}
	return nil
}

func (dao *LocalStateDAO) AddUser(username string, passwordHash string) error {
	_, err := dao.db.Exec(constants.INSERT_USER, username, passwordHash)
	if err != nil {
		return fmt.Errorf("could not add user: %w", err)
	}
	return nil
}

func (dao *LocalStateDAO) GetUserByUsername(username string) (*structs.User, error) {
	var u structs.User
	err := dao.db.QueryRow(constants.SELECT_USER_BY_USERNAME, username).Scan(&u.ID, &u.Username, &u.PasswordHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not get user by username: %w", err)
	}
	return &u, nil
}

func (dao *LocalStateDAO) GetUserCount() (int, error) {
	var count int
	err := dao.db.QueryRow(constants.COUNT_USERS).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("could not get user count: %w", err)
	}
	return count, nil
}
