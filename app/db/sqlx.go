package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

func (d *Database) NamedSelectContext(ctx context.Context, dest interface{}, query string, arg interface{}) error {
	q, args, err := sqlx.BindNamed(sqlx.BindType(d.Sqlx.DriverName()), query, arg)
	if err != nil {
		return err
	}

	return d.Sqlx.SelectContext(ctx, dest, q, args...)
}

func (d *Database) NamedGetContext(ctx context.Context, dest interface{}, query string, arg interface{}) error {
	q, args, err := sqlx.BindNamed(sqlx.BindType(d.Sqlx.DriverName()), query, arg)
	if err != nil {
		return err
	}

	return d.Sqlx.GetContext(ctx, dest, q, args...)
}

func (d *Database) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	q, args, err := sqlx.BindNamed(sqlx.BindType(d.Sqlx.DriverName()), query, arg)
	if err != nil {
		return nil, err
	}

	return d.Sqlx.ExecContext(ctx, q, args...)
}

func (d *Database) NamedExecContextReturnID(ctx context.Context, query string, arg interface{}, ID interface{}) error {
	q, args, err := sqlx.BindNamed(sqlx.BindType(d.Sqlx.DriverName()), query, arg)
	if err != nil {
		return err
	}

	return d.Sqlx.QueryRowx(q, args...).Scan(ID)
}

func (d *Database) NamedExecContextReturnObj(ctx context.Context, query string, arg interface{}, obj interface{}) error {
	q, args, err := sqlx.BindNamed(sqlx.BindType(d.Sqlx.DriverName()), query, arg)
	if err != nil {
		return err
	}

	return d.Sqlx.QueryRowx(q, args...).StructScan(obj)
}

func stringSliceContains(slice []string, target string) bool {
	for _, element := range slice {
		if element == target {
			return true
		}
	}
	return false
}

func (d *Database) GetFormattedColumnNames(columns []string, excludeColumns ...string) string {
	if len(columns) == 0 {
		return ""
	}

	var formattedColumns []string

	for _, column := range columns {
		// Assuming column names are safe and don't need escaping
		if !stringSliceContains(excludeColumns, column) {
			formattedColumns = append(formattedColumns, fmt.Sprintf("%s = :%s", column, column))
		}
	}

	return strings.Join(formattedColumns, ", ")
}

func (d *Database) GetStringMapKeys(goMap map[string]interface{}) []string {
	keys := make([]string, 0, len(goMap))

	for key := range goMap {
		keys = append(keys, key)
	}

	return keys
}
