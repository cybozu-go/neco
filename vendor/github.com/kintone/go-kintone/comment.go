package kintone

import (
	"encoding/json"
	"errors"
)

//
const (
	ConstCommentMentionTypeGroup      = "GROUP"
	ConstCommentMentionTypeDepartment = "ORGANIZATION"
	ConstCommentMentionTypeUser       = "USER"
)

// ObjMentions structure
type ObjMention struct {
	Code string `json:"code"`
	Type string `json:"type"`
}

// ObjCreator structure
type ObjCreator struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// Comment structure
type Comment struct {
	Id        string        `json:"id"`
	Text      string        `json:"text"`
	CreatedAt string        `json:"createdAt"`
	Creator   *ObjCreator   `json:"creator"`
	Mentions  []*ObjMention `json:"mentions"`
}

// DecodeRecordComments decodes JSON response for comment api
func DecodeRecordComments(b []byte) ([]Comment, error) {
	var comments struct {
		MyComments []Comment `json:"comments"`
		Older      bool      `json:"older"`
		Newer      bool      `json:"newer"`
	}
	err := json.Unmarshal(b, &comments)
	if err != nil {
		return nil, errors.New("Invalid JSON format")
	}
	return comments.MyComments, nil
}
