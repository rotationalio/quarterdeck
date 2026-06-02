package fields

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONB json.RawMessage

type NullJSONB struct {
	Valid bool
	JSONB JSONB
}

var JSONNull = []byte("null")

//============================================================================
// JSONB Methods
//============================================================================

func (j *JSONB) Scan(src any) error {
	if src == nil {
		*j = nil
	}

	switch src := src.(type) {
	case []byte:
		*j = append((*j)[0:0], src...)
	case string:
		*j = append((*j)[0:0], src...)
	default:
		return fmt.Errorf("cannot scan type %T into JSONB", src)
	}

	// If the JSONB value is null then set the value to nil
	if bytes.Equal([]byte(*j), JSONNull) {
		*j = nil
	}
	return nil
}

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j JSONB) IsNull() bool {
	return len(j) == 0 || bytes.Equal(j, JSONNull)
}

func (j JSONB) UnmarshalTo(dst any) error {
	if len(j) == 0 {
		return nil
	}
	return json.Unmarshal(j, dst)
}

func (j *JSONB) MarshalFrom(src any) (err error) {
	if src == nil {
		*j = nil
		return nil
	}

	var data []byte
	if data, err = json.Marshal(src); err != nil {
		return err
	}

	*j = append((*j)[0:0], data...)
	return nil
}

//============================================================================
// NullJSONB Methods
//============================================================================

func (n *NullJSONB) Scan(src any) (err error) {
	if src == nil {
		n.JSONB, n.Valid = nil, false
		return nil
	}

	if err = n.JSONB.Scan(src); err != nil {
		n.Valid = false
		return err
	}

	n.Valid = !n.JSONB.IsNull()
	return nil
}

func (n NullJSONB) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.JSONB.Value()
}

func (j NullJSONB) UnmarshalTo(dst any) error {
	if !j.Valid {
		return nil
	}
	return j.JSONB.UnmarshalTo(dst)
}

func (j *NullJSONB) MarshalFrom(src any) (err error) {
	if err = j.JSONB.MarshalFrom(src); err != nil {
		j.Valid = false
		return err
	}

	if j.JSONB.IsNull() {
		j.JSONB = nil
		j.Valid = false
	} else {
		j.Valid = true
	}

	return nil
}
