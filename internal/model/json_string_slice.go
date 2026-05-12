package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringSliceJSON — JSONB-массив строк в PostgreSQL; драйвер отдаёт []byte, обычный Scan в []string не подходит.
type StringSliceJSON []string

// Scan реализует sql.Scanner.
func (s *StringSliceJSON) Scan(src interface{}) error {
	if src == nil {
		*s = nil
		return nil
	}
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("StringSliceJSON: unsupported src type %T", src)
	}
	if len(raw) == 0 {
		*s = StringSliceJSON{}
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	*s = StringSliceJSON(out)
	return nil
}

// Value реализует driver.Valuer (для параметров INSERT/UPDATE при необходимости).
func (s StringSliceJSON) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(s))
}
