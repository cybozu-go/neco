// (C) 2014 Cybozu.  All rights reserved.
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file.

package kintone

import (
	"encoding/json"
	"strconv"
	"time"
)

// Field type identifiers.
const (
	FT_SINGLE_LINE_TEXT = "SINGLE_LINE_TEXT"
	FT_MULTI_LINE_TEXT  = "MULTI_LINE_TEXT"
	FT_RICH_TEXT        = "RICH_TEXT"
	FT_DECIMAL          = "NUMBER"
	FT_CALC             = "CALC"
	FT_CHECK_BOX        = "CHECK_BOX"
	FT_RADIO            = "RADIO_BUTTON"
	FT_SINGLE_SELECT    = "DROP_DOWN"
	FT_MULTI_SELECT     = "MULTI_SELECT"
	FT_FILE             = "FILE"
	FT_LINK             = "LINK"
	FT_DATE             = "DATE"
	FT_TIME             = "TIME"
	FT_DATETIME         = "DATETIME"
	FT_USER             = "USER_SELECT"
	FT_ORGANIZATION     = "ORGANIZATION_SELECT"
	FT_GROUP            = "GROUP_SELECT"
	FT_CATEGORY         = "CATEGORY"
	FT_STATUS           = "STATUS"
	FT_ASSIGNEE         = "STATUS_ASSIGNEE"
	FT_RECNUM           = "RECORD_NUMBER"
	FT_CREATOR          = "CREATOR"
	FT_CTIME            = "CREATED_TIME"
	FT_MODIFIER         = "MODIFIER"
	FT_MTIME            = "UPDATED_TIME"
	FT_SUBTABLE         = "SUBTABLE"
	FT_ID               = "__ID__"
	FT_REVISION         = "__REVISION__"
)

// SingleLineTextField is a field type for single-line texts.
type SingleLineTextField string
func (f SingleLineTextField) JSONValue() (interface{}) {
	return string(f);
}
func (f SingleLineTextField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_SINGLE_LINE_TEXT,
		"value": f.JSONValue(),
	})
}

// MultiLineTextField is a field type for multi-line texts.
type MultiLineTextField string
func (f MultiLineTextField) JSONValue() (interface{}) {
	return string(f);
}
func (f MultiLineTextField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_MULTI_LINE_TEXT,
		"value": f.JSONValue(),
	})
}

// RichTextField is a field type for HTML rich texts.
type RichTextField string
func (f RichTextField) JSONValue() (interface{}) {
	return string(f);
}
func (f RichTextField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_RICH_TEXT,
		"value": f.JSONValue(),
	})
}

// DecimalField is a field type for decimal numbers.
type DecimalField string
func (f DecimalField) JSONValue() (interface{}) {
	return string(f);
}
func (f DecimalField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_DECIMAL,
		"value": f.JSONValue(),
	})
}

// CalcField is a field type for auto-calculated values.
type CalcField string
func (f CalcField) JSONValue() (interface{}) {
	return string(f);
}
func (f CalcField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_CALC,
		"value": f.JSONValue(),
	})
}

// CheckBoxField is a field type for selected values in a check-box.
type CheckBoxField []string
func (f CheckBoxField) JSONValue() (interface{}) {
	return []string(f);
}
func (f CheckBoxField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_CHECK_BOX,
		"value": f.JSONValue(),
	})
}

// RadioButtonField is a field type for the selected value by a radio-button.
type RadioButtonField string
func (f RadioButtonField) JSONValue() (interface{}) {
	return string(f);
}
func (f RadioButtonField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_RADIO,
		"value": f.JSONValue(),
	})
}

// SingleSelectField is a field type for the selected value in a selection box.
type SingleSelectField struct {
	String string // Selected value.
	Valid  bool   // If not selected, false.
}
func (f SingleSelectField) JSONValue() (interface{}) {
	if f.Valid {
		return f.String
	} else {
		return nil
	}
}
func (f SingleSelectField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_SINGLE_SELECT,
		"value": f.JSONValue(),
	})
}

// MultiSelectField is a field type for selected values in a selection box.
type MultiSelectField []string
func (f MultiSelectField) JSONValue() (interface{}) {
	return []string(f);
}
func (f MultiSelectField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_MULTI_SELECT,
		"value": f.JSONValue(),
	})
}

// File is a struct for an uploaded file.
type File struct {
	ContentType string `json:"contentType"` // MIME type of the file
	FileKey     string `json:"fileKey"`     // BLOB ID of the file
	Name        string `json:"name"`        // File name
	Size        uint64 `json:"size,string"` // The file size
}
func (f *File) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		map[string]interface{}{
			"contentType": f.ContentType,
			"fileKey":     f.FileKey,
			"name":        f.Name,
			"size":        strconv.FormatUint(f.Size, 10),
		})
}

// FileField is a field type for uploaded files.
type FileField []File
func (f FileField) JSONValue() (interface{}) {
	return []File(f);
}
func (f FileField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_FILE,
		"value": f.JSONValue(),
	})
}

// LinkField is a field type for hyper-links.
type LinkField string
func (f LinkField) JSONValue() (interface{}) {
	return string(f);
}
func (f LinkField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_LINK,
		"value": f.JSONValue(),
	})
}

// DateField is a field type for dates.
type DateField struct {
	Date  time.Time // stores date information.
	Valid bool      // false when not set.
}

// NewDateField returns an instance of DateField.
func NewDateField(year int, month time.Month, day int) DateField {
	return DateField{
		time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
		true,
	}
}
func (f DateField) JSONValue() (interface{}) {
	if f.Valid {
		return f.Date.Format("2006-01-02");
	} else {
		return nil;
	}
}
func (f DateField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_DATE,
		"value": f.JSONValue(),
	})
}

// TimeField is a field type for times.
type TimeField struct {
	Time  time.Time // stores time information.
	Valid bool      // false when not set.
}

// NewTimeField returns an instance of TimeField.
func NewTimeField(hour, min int) TimeField {
	return TimeField{
		time.Date(1, time.January, 1, hour, min, 0, 0, time.UTC),
		true,
	}
}
func (f TimeField) JSONValue() (interface{}) {
	if f.Valid {
		return f.Time.Format("15:04:05");
	} else {
		return nil;
	}
}
func (f TimeField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_TIME,
		"value": f.JSONValue(),
	})
}

// DateTimeField is a field type for date & time.
type DateTimeField struct {
	Time  time.Time // stores time information.
	Valid bool      // false when not set.
}

// NewDateTimeField returns an instance of DateTimeField.
func NewDateTimeField(year int, month time.Month, day, hour, min int) DateTimeField {
	return DateTimeField{
		time.Date(year, month, day, hour, min, 0, 0, time.UTC),
		true,
	}
}
func (f DateTimeField) JSONValue() (interface{}) {
	if f.Valid {
		return f.Time.Format(time.RFC3339);
	} else {
		return nil;
	}
}
func (f DateTimeField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_DATETIME,
		"value": f.JSONValue(),
	})
}

// User represents a user entry.
type User struct {
	Code string `json:"code"` // A unique identifer of the user.
	Name string `json:"name"` // The user name.
}

// UserField is a field type for user entries.
type UserField []User
func (f UserField) JSONValue() (interface{}) {
	return []User(f);
}
func (f UserField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_USER,
		"value": f.JSONValue(),
	})
}

// Organization represents a department entry.
type Organization struct {
	Code string `json:"code"` // A unique identifer of the department.
	Name string `json:"name"` // The department name.
}

// OrganizationField is a field type for department entries.
type OrganizationField []Organization
func (f OrganizationField) JSONValue() (interface{}) {
	return []Organization(f);
}
func (f OrganizationField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_ORGANIZATION,
		"value": f.JSONValue(),
	})
}

// Group represents a group(or role) entry.
type Group struct {
	Code string `json:"code"` // A unique identifer of the group(or role).
	Name string `json:"name"` // The group name.
}

// GroupField is a field type for group(or role) entries.
type GroupField []Group
func (f GroupField) JSONValue() (interface{}) {
	return []Group(f);
}
func (f GroupField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_GROUP,
		"value": f.JSONValue(),
	})
}

// CategoryField is a list of category names.
type CategoryField []string
func (f CategoryField) JSONValue() (interface{}) {
	return []string(f);
}
func (f CategoryField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_CATEGORY,
		"value": f.JSONValue(),
	})
}

// StatusField is a string label of a record status.
type StatusField string
func (f StatusField) JSONValue() (interface{}) {
	return string(f);
}
func (f StatusField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_STATUS,
		"value": f.JSONValue(),
	})
}

// AssigneeField is a list of user entries who are assigned to a record.
type AssigneeField []User
func (f AssigneeField) JSONValue() (interface{}) {
	return []User(f);
}
func (f AssigneeField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_ASSIGNEE,
		"value": f.JSONValue(),
	})
}

// RecordNumberField is a record number.
type RecordNumberField string
func (f RecordNumberField) JSONValue() (interface{}) {
	return string(f);
}
func (f RecordNumberField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_RECNUM,
		"value": f.JSONValue(),
	})
}

// CreatorField is a user who created a record.
type CreatorField User
func (f CreatorField) JSONValue() (interface{}) {
	return User(f);
}
func (f CreatorField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_CREATOR,
		"value": f.JSONValue(),
	})
}

// CreationTimeField is the time when a record is created.
type CreationTimeField time.Time
func (t CreationTimeField) JSONValue() (interface{}) {
	return time.Time(t).Format(time.RFC3339);
}
func (t CreationTimeField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_CTIME,
		"value": t.JSONValue(),
	})
}

// ModifierField is a user who modified a record last.
type ModifierField User
func (f ModifierField) JSONValue() (interface{}) {
		return User(f);
}
func (f ModifierField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_MODIFIER,
		"value": f.JSONValue(),
	})
}

// ModificationTimeField is the time when a record is last modified.
type ModificationTimeField time.Time
func (t ModificationTimeField) JSONValue() (interface{}) {
		return time.Time(t).Format(time.RFC3339);
}
func (t ModificationTimeField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_MTIME,
		"value": t.JSONValue(),
	})
}

// SubTableEntry is a type for an entry in a subtable.
type SubTableEntry struct {
	Id    string                 `json:"id"`    // The entry ID
	Value map[string]interface{} `json:"value"` // Subtable data fields.
}

// SubTableField is a list of subtable entries.
type SubTableField []*Record
func (f SubTableField) JSONValue() (interface{}) {
	type sub_record struct {
		Record *Record `json:"value"`
	}
	type sub_record_with_id struct {
		Id    uint64   `json:"id,string"`
		Record *Record `json:"value"`
	}
	recs := make([]interface{}, 0, len(f))
	for _, rec := range f {
		if (rec.id == 0) {
			recs = append(recs, sub_record{rec})
		} else {
			recs = append(recs, sub_record_with_id{rec.id, rec})
		}
	}
	return recs;
}
func (f SubTableField) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  FT_SUBTABLE,
		"value": f.JSONValue(),
	})
}

// IsBuiltinField returns true if the field is a built-in field.
func IsBuiltinField(o interface{}) bool {
	switch o.(type) {
	case CalcField:
		return true
	case CategoryField:
		return true
	case StatusField:
		return true
	case AssigneeField:
		return true
	case RecordNumberField:
		return true
	case CreatorField:
		return true
	case CreationTimeField:
		return true
	case ModifierField:
		return true
	case ModificationTimeField:
		return true
	}
	return false
}
