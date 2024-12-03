package database

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/types"
)

func (s *service) SaveIperfServer(ctx context.Context, name, host string, port int) (*types.SavedIperfServer, error) {
	query := s.sqlBuilder.
		Insert("saved_iperf_servers").
		Columns("name", "host", "port").
		Values(name, host, port).
		Suffix("RETURNING id, name, host, port, created_at, updated_at")

	server := new(types.SavedIperfServer)
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(&server.ID, &server.Name, &server.Host, &server.Port, &server.CreatedAt, &server.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (s *service) GetIperfServers(ctx context.Context) ([]types.SavedIperfServer, error) {
	query := s.sqlBuilder.
		Select("id", "name", "host", "port", "created_at", "updated_at").
		From("saved_iperf_servers").
		OrderBy("created_at DESC")

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []types.SavedIperfServer
	for rows.Next() {
		var server types.SavedIperfServer
		err := rows.Scan(&server.ID, &server.Name, &server.Host, &server.Port, &server.CreatedAt, &server.UpdatedAt)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return servers, nil
}

func (s *service) DeleteIperfServer(ctx context.Context, id int) error {
	query := s.sqlBuilder.
		Delete("saved_iperf_servers").
		Where(sq.Eq{"id": id})

	result, err := query.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
