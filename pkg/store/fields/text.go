package fields

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type StringArray []string

type NullStringArray struct {
	Valid       bool
	StringArray StringArray
}

//============================================================================
// StringArray Methods
//============================================================================

func (s *StringArray) Scan(src any) (err error) {
	if src == nil {
		*s = nil
	}

	switch src := src.(type) {
	case []byte:
		if err = json.Unmarshal(src, s); err != nil {
			return err
		}
	case string:
		if err = json.Unmarshal([]byte(src), s); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot scan type %T into StringArray", src)
	}

	return nil
}

func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return json.Marshal(s)
}

//============================================================================
// NullStringArray Methods
//============================================================================

func (n *NullStringArray) Scan(src any) (err error) {
	if src == nil {
		n.StringArray, n.Valid = nil, false
		return nil
	}

	if err = n.StringArray.Scan(src); err != nil {
		n.Valid = false
		return err
	}

	return nil
}

func (n NullStringArray) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.StringArray.Value()
}
