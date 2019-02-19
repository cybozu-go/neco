// (C) 2014 Cybozu.  All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file.

package kintone

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// Record represens a record in an application.
//
// Fields is a mapping between field IDs and fields.
// Although field types are shown as interface{}, they are guaranteed
// to be one of a *Field type in this package.
type Record struct {
	id       uint64
	revision int64
	Fields   map[string]interface{}
}

// NewRecord creates an instance of Record.
//
// The revision number is initialized to -1.
func NewRecord(fields map[string]interface{}) *Record {
	return &Record{0, -1, fields}
}

// NewRecord creates using an existing record id.
//
// The revision number is initialized to -1.
func NewRecordWithId(id uint64, fields map[string]interface{}) *Record {
	return &Record{id, -1, fields}
}

// MarshalJSON marshals field data of a record into JSON.
func (rec Record) MarshalJSON() ([]byte, error) {
	return json.Marshal(rec.Fields)
}

// Id returns the record number.
//
// A record number is unique within an application.
func (rec Record) Id() uint64 {
	return rec.id
}

// Revision returns the record revision number.
func (rec Record) Revision() int64 {
	return rec.revision
}

// Assert string list.
func stringList(l []interface{}) []string {
	sl := make([]string, len(l))
	for i, v := range l {
		sl[i] = v.(string)
	}
	return sl
}

// Convert user list.
func userList(l []interface{}) ([]User, error) {
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	var ul []User
	err = json.Unmarshal(b, &ul)
	if err != nil {
		return nil, err
	}
	return ul, nil
}

// Convert organization list.
func organizationList(l []interface{}) ([]Organization, error) {
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	var ol []Organization
	err = json.Unmarshal(b, &ol)
	if err != nil {
		return nil, err
	}
	return ol, nil
}

// Convert group list.
func groupList(l []interface{}) ([]Group, error) {
	b, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	var gl []Group
	err = json.Unmarshal(b, &gl)
	if err != nil {
		return nil, err
	}
	return gl, nil
}

// Convert string "record number" into an integer.
func numericId(id string) (uint64, error) {
	n := strings.LastIndex(id, "-")
	if n != -1 {
		id = id[(n + 1):]
	}
	nid, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, err
	}
	return nid, nil
}

type recordData map[string]struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func decodeRecordData(data recordData) (*Record, error) {
	fields := make(map[string]interface{})
	rec := &Record{0, -1, fields}
	for key, v := range data {
		switch v.Type {
		case FT_SINGLE_LINE_TEXT:
			fields[key] = SingleLineTextField(v.Value.(string))
		case FT_MULTI_LINE_TEXT:
			fields[key] = MultiLineTextField(v.Value.(string))
		case FT_RICH_TEXT:
			fields[key] = RichTextField(v.Value.(string))
		case FT_DECIMAL:
			fields[key] = DecimalField(v.Value.(string))
		case FT_CALC:
			fields[key] = CalcField(v.Value.(string))
		case FT_CHECK_BOX:
			fields[key] = CheckBoxField(stringList(v.Value.([]interface{})))
		case FT_RADIO:
			if v.Value == nil {
				fields[key] = RadioButtonField("")
			} else {
				fields[key] = RadioButtonField(v.Value.(string))
			}
		case FT_SINGLE_SELECT:
			if v.Value == nil {
				fields[key] = SingleSelectField{Valid: false}
			} else {
				fields[key] = SingleSelectField{v.Value.(string), true}
			}
		case FT_MULTI_SELECT:
			fields[key] = MultiSelectField(stringList(v.Value.([]interface{})))
		case FT_FILE:
			b1, err := json.Marshal(v.Value)
			if err != nil {
				return nil, err
			}
			var fl []File
			err = json.Unmarshal(b1, &fl)
			if err != nil {
				return nil, err
			}
			fields[key] = FileField(fl)
		case FT_LINK:
			fields[key] = LinkField(v.Value.(string))
		case FT_DATE:
			if v.Value == nil {
				fields[key] = DateField{Valid: false}
			} else {
				d, err := time.Parse("2006-01-02", v.Value.(string))
				if err != nil {
					return nil, fmt.Errorf("Invalid date: %v", v.Value)
				}
				fields[key] = DateField{d, true}
			}
		case FT_TIME:
			if v.Value == nil {
				fields[key] = TimeField{Valid: false}
			} else {
				t, err := time.Parse("15:04", v.Value.(string))
				if err != nil {
					t, err = time.Parse("15:04:05", v.Value.(string))
					if err != nil {
						return nil, fmt.Errorf("Invalid time: %v", v.Value)
					}
				}
				fields[key] = TimeField{t, true}
			}
		case FT_DATETIME:
			if v.Value == "" {
				fields[key] = DateTimeField{Valid: false}
			} else {
				if s, ok := v.Value.(string); ok {
					dt, err := time.Parse(time.RFC3339, s)
					if err != nil {
						return nil, fmt.Errorf("Invalid datetime: %v", v.Value)
					}
					fields[key] = DateTimeField{dt, true}
				}
			}
		case FT_USER:
			ul, err := userList(v.Value.([]interface{}))
			if err != nil {
				return nil, err
			}
			fields[key] = UserField(ul)
		case FT_ORGANIZATION:
			ol, err := organizationList(v.Value.([]interface{}))
			if err != nil {
				return nil, err
			}
			fields[key] = OrganizationField(ol)
		case FT_GROUP:
			gl, err := groupList(v.Value.([]interface{}))
			if err != nil {
				return nil, err
			}
			fields[key] = GroupField(gl)
		case FT_CATEGORY:
			fields[key] = CategoryField(stringList(v.Value.([]interface{})))
		case FT_STATUS:
			fields[key] = StatusField(v.Value.(string))
		case FT_ASSIGNEE:
			al, err := userList(v.Value.([]interface{}))
			if err != nil {
				return nil, err
			}
			fields[key] = AssigneeField(al)
		case FT_RECNUM:
			if nid, err := numericId(v.Value.(string)); err != nil {
				return nil, err
			} else {
				rec.id = nid
			}
			fields[key] = RecordNumberField(v.Value.(string))
		case FT_CREATOR:
			creator := v.Value.(map[string]interface{})
			fields[key] = CreatorField{
				creator["code"].(string),
				creator["name"].(string),
			}
		case FT_CTIME:
			var ctime time.Time
			if ctime.UnmarshalText([]byte(v.Value.(string))) != nil {
				return nil, fmt.Errorf("Invalid datetime: %v", v.Value)
			}
			fields[key] = CreationTimeField(ctime)
		case FT_MODIFIER:
			modifier := v.Value.(map[string]interface{})
			fields[key] = ModifierField{
				modifier["code"].(string),
				modifier["name"].(string),
			}
		case FT_MTIME:
			var mtime time.Time
			if mtime.UnmarshalText([]byte(v.Value.(string))) != nil {
				return nil, fmt.Errorf("Invalid datetime: %v", v.Value)
			}
			fields[key] = CreationTimeField(mtime)
		case FT_SUBTABLE:
			b2, err := json.Marshal(v.Value)
			if err != nil {
				return nil, err
			}
			var stl []SubTableEntry
			err = json.Unmarshal(b2, &stl)
			if err != nil {
				return nil, err
			}
			ra := make([]*Record, len(stl))
			for i, sr := range stl {
				b3, err := json.Marshal(sr.Value)
				if err != nil {
					return nil, err
				}
				var rd recordData
				err = json.Unmarshal(b3, &rd)
				if err != nil {
					return nil, err
				}
				r, err := decodeRecordData(recordData(rd))
				if err != nil {
					return nil, err
				}
				id, err := strconv.ParseUint(sr.Id, 10, 64)
				if err != nil {
					return nil, err
				}
				r.id = id
				ra[i] = r
			}
			fields[key] = SubTableField(ra)
		case FT_ID:
			id, err := strconv.ParseUint(v.Value.(string), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("Invalid record ID: %v", v.Value)
			}
			rec.id = id
		case FT_REVISION:
			revision, err := strconv.ParseInt(v.Value.(string), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("Invalid revision number: %v", v.Value)
			}
			rec.revision = revision
		default:
			log.Printf("Unknown type: %v", v.Type)
		}
	}
	return rec, nil
}

// DecodeRecords decodes JSON response for multi-get API.
func DecodeRecords(b []byte) ([]*Record, error) {
	var t struct {
		Records []recordData `json:"records"`
	}
	err := json.Unmarshal(b, &t)
	if err != nil {
		return nil, errors.New("Invalid JSON format")
	}
	rec_list := make([]*Record, len(t.Records))
	for i, rd := range t.Records {
		r, err := decodeRecordData(rd)
		if err != nil {
			return nil, err
		}
		rec_list[i] = r
	}
	return rec_list, nil
}

// DecodeRecord decodes JSON response for single-get API.
func DecodeRecord(b []byte) (*Record, error) {
	var t struct {
		RecordData recordData `json:"record"`
	}
	err := json.Unmarshal(b, &t)
	if err != nil {
		return nil, errors.New("Invalid JSON format")
	}
	rec, err := decodeRecordData(t.RecordData)
	if err != nil {
		return nil, err
	}
	return rec, nil
}
