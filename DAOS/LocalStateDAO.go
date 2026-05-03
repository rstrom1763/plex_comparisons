package DAOS

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rstrom1763/plex_comparisons/constants"
	"github.com/rstrom1763/plex_comparisons/structs"
)

type LocalStateDAO struct {
	db *sql.DB
}

func NewLocalStateDAO(dbPath string) (*LocalStateDAO, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open local state database: %w", err)
	}

	if _, err := db.Exec(constants.CREATE_SERVERS_TABLE); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("could not create servers table: %w", err)
	}

	if _, err := db.Exec(constants.CREATE_SAVED_FILTERS_TABLE); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("could not create saved filters table: %w", err)
	}

	return &LocalStateDAO{db: db}, nil
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
