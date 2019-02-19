/*
Package kintone provides interfaces for kintone REST API.

See http://developers.kintone.com/ for API specs.

	import (
		"log"

		"github.com/kintone/go-kintone"
	)
	...
	app := &kintone.App{
		"example.cybozu.com",
		"user1",
		"password",
		25,
	}

To retrieve 3 records from a kintone app (id=25):
	records, err := app.GetRecords(nil, "limit 3")
	if err != nil {
		log.Fatal(err)
	}
	// use records

To retrieve 10 latest comments in record (id=3) from a kintone app (id=25)
	var offset uint64 = 0
	var limit uint64 = 10
	comments, err := app.GetRecordComments(3, "desc", offset, limit)
	if err != nil {
		log.Fatal(err)
	}
	// use comments

To retrieve oldest 10 comments and skips the first 30 comments in record (id=3) from a kintone app (id=25)
	var offset uint64 = 30
	var limit uint64 = 10
	comments, err := app.GetRecordComments(3, "asc", offset, limit)
	if err != nil {
		log.Fatal(err)
	}
	// use comments

To add comments into record (id=3) from a kintone app (id=25)
	mentionMemberCybozu := &ObjMention{Code: "cybozu", Type: kintone.ConstCommentMentionTypeUser}
	mentionGroupAdmin := &ObjMention{Code: "Administrators", Type: kintone.ConstCommentMentionTypeGroup}
	mentionDepartmentAdmin := &ObjMention{Code: "Admin", Type: ConstCommentMentionTypeDepartment}

	var cmt Comment
	cmt.Text = "Test comment 222"
	cmt.Mentions = []*ObjMention{mentionGroupAdmin, mentionMemberCybozu, mentionDepartmentAdmin}
	cmtID, err := app.AddRecordComment(3, &cmt)
	if err != nil {
		log.Fatal(err)
	}
	// use comments id

To remove comments (id=12) in the record (id=3) from a kintone app (id=25)
	err := app.DeleteComment(3, 12)

	if err != nil {
		log.Fatal(err)
	}

*/
package kintone
