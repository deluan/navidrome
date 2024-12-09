package criteria_test

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/navidrome/navidrome/model/criteria"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("Operators", func() {
	rangeStart := Date(time.Date(2021, 10, 01, 0, 0, 0, 0, time.Local))
	rangeEnd := Date(time.Date(2021, 11, 01, 0, 0, 0, 0, time.Local))

	DescribeTable("ToSQL",
		func(op Expression, expectedSql string, expectedArgs ...any) {
			sql, args, err := op.ToSql()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(sql).To(gomega.Equal(expectedSql))
			gomega.Expect(args).To(gomega.HaveExactElements(expectedArgs...))
		},
		Entry("is [string]", Is{"title": "Low Rider"}, "media_file.title = ?", "Low Rider"),
		Entry("is [bool]", Is{"loved": true}, "COALESCE(annotation.starred, false) = ?", true),
		Entry("isNot", IsNot{"title": "Low Rider"}, "media_file.title <> ?", "Low Rider"),
		Entry("gt", Gt{"playCount": 10}, "COALESCE(annotation.play_count, 0) > ?", 10),
		Entry("lt", Lt{"playCount": 10}, "COALESCE(annotation.play_count, 0) < ?", 10),
		Entry("contains", Contains{"title": "Low Rider"}, "media_file.title LIKE ?", "%Low Rider%"),
		Entry("notContains", NotContains{"title": "Low Rider"}, "media_file.title NOT LIKE ?", "%Low Rider%"),
		Entry("startsWith", StartsWith{"title": "Low Rider"}, "media_file.title LIKE ?", "Low Rider%"),
		Entry("endsWith", EndsWith{"title": "Low Rider"}, "media_file.title LIKE ?", "%Low Rider"),
		Entry("inTheRange [number]", InTheRange{"year": []int{1980, 1990}}, "(media_file.year >= ? AND media_file.year <= ?)", 1980, 1990),
		Entry("inTheRange [date]", InTheRange{"lastPlayed": []Date{rangeStart, rangeEnd}}, "(annotation.play_date >= ? AND annotation.play_date <= ?)", rangeStart, rangeEnd),
		Entry("before", Before{"lastPlayed": rangeStart}, "annotation.play_date < ?", rangeStart),
		Entry("after", After{"lastPlayed": rangeStart}, "annotation.play_date > ?", rangeStart),

		// InPlaylis and NotInPlaylist are special cases
		Entry("inPlaylist", InPlaylist{"id": "deadbeef-dead-beef"}, "media_file.id IN "+
			"(SELECT media_file_id FROM playlist_tracks pl LEFT JOIN playlist on pl.playlist_id = playlist.id WHERE (pl.playlist_id = ? AND playlist.public = ?))", "deadbeef-dead-beef", 1),
		Entry("notInPlaylist", NotInPlaylist{"id": "deadbeef-dead-beef"}, "media_file.id NOT IN "+
			"(SELECT media_file_id FROM playlist_tracks pl LEFT JOIN playlist on pl.playlist_id = playlist.id WHERE (pl.playlist_id = ? AND playlist.public = ?))", "deadbeef-dead-beef", 1),

		// TODO These may be flaky
		Entry("inTheLast", InTheLast{"lastPlayed": 30}, "annotation.play_date > ?", StartOfPeriod(30, time.Now())),
		Entry("notInTheLast", NotInTheLast{"lastPlayed": 30}, "(annotation.play_date < ? OR annotation.play_date IS NULL)", StartOfPeriod(30, time.Now())),

		// Tag tests
		Entry("tag is [string]", Is{"genre": "Rock"}, "exists (select 1 from json_tree(tags, '$.genre') where key='value' and value = ?)", "Rock"),
		Entry("tag isNot [string]", IsNot{"genre": "Rock"}, "not exists (select 1 from json_tree(tags, '$.genre') where key='value' and value = ?)", "Rock"),
		Entry("tag gt", Gt{"genre": "A"}, "exists (select 1 from json_tree(tags, '$.genre') where key='value' and value > ?)", "A"),
		Entry("tag lt", Lt{"genre": "Z"}, "exists (select 1 from json_tree(tags, '$.genre') where key='value' and value < ?)", "Z"),
		Entry("tag contains", Contains{"genre": "Rock"}, "exists (select 1 from json_tree(tags, '$.genre') where key='value' and value LIKE ?)", "%Rock%"),
		Entry("tag not contains", NotContains{"genre": "Rock"}, "not exists (select 1 from json_tree(tags, '$.genre') where key='value' and value LIKE ?)", "%Rock%"),
		Entry("tag startsWith", StartsWith{"genre": "Soft"}, "exists (select 1 from json_tree(tags, '$.genre') where key='value' and value LIKE ?)", "Soft%"),
		Entry("tag endsWith", EndsWith{"genre": "Rock"}, "exists (select 1 from json_tree(tags, '$.genre') where key='value' and value LIKE ?)", "%Rock"),
	)

	Describe("Custom Tags", func() {
		It("generates valid SQL", func() {
			AddTagNames([]string{"mood"})
			op := EndsWith{"mood": "Soft"}
			sql, args, err := op.ToSql()
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(sql).To(gomega.Equal("exists (select 1 from json_tree(tags, '$.mood') where key='value' and value LIKE ?)"))
			gomega.Expect(args).To(gomega.HaveExactElements("%Soft"))
		})
		It("returns errors for unknown tag names", func() {
			op := EndsWith{"unknown": "value"}
			sql, args, _ := op.ToSql()
			gomega.Expect(sql).To(gomega.BeEmpty())
			gomega.Expect(args).To(gomega.BeEmpty())
		})
	})

	DescribeTable("JSON Marshaling",
		func(op Expression, jsonString string) {
			obj := And{op}
			newJs, err := json.Marshal(obj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(string(newJs)).To(gomega.Equal(fmt.Sprintf(`{"all":[%s]}`, jsonString)))

			var unmarshalObj UnmarshalConjunctionType
			js := "[" + jsonString + "]"
			err = json.Unmarshal([]byte(js), &unmarshalObj)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(unmarshalObj[0]).To(gomega.Equal(op))
		},
		Entry("is [string]", Is{"title": "Low Rider"}, `{"is":{"title":"Low Rider"}}`),
		Entry("is [bool]", Is{"loved": false}, `{"is":{"loved":false}}`),
		Entry("isNot", IsNot{"title": "Low Rider"}, `{"isNot":{"title":"Low Rider"}}`),
		Entry("gt", Gt{"playCount": 10.0}, `{"gt":{"playCount":10}}`),
		Entry("lt", Lt{"playCount": 10.0}, `{"lt":{"playCount":10}}`),
		Entry("contains", Contains{"title": "Low Rider"}, `{"contains":{"title":"Low Rider"}}`),
		Entry("notContains", NotContains{"title": "Low Rider"}, `{"notContains":{"title":"Low Rider"}}`),
		Entry("startsWith", StartsWith{"title": "Low Rider"}, `{"startsWith":{"title":"Low Rider"}}`),
		Entry("endsWith", EndsWith{"title": "Low Rider"}, `{"endsWith":{"title":"Low Rider"}}`),
		Entry("inTheRange [number]", InTheRange{"year": []any{1980.0, 1990.0}}, `{"inTheRange":{"year":[1980,1990]}}`),
		Entry("inTheRange [date]", InTheRange{"lastPlayed": []any{"2021-10-01", "2021-11-01"}}, `{"inTheRange":{"lastPlayed":["2021-10-01","2021-11-01"]}}`),
		Entry("before", Before{"lastPlayed": "2021-10-01"}, `{"before":{"lastPlayed":"2021-10-01"}}`),
		Entry("after", After{"lastPlayed": "2021-10-01"}, `{"after":{"lastPlayed":"2021-10-01"}}`),
		Entry("inTheLast", InTheLast{"lastPlayed": 30.0}, `{"inTheLast":{"lastPlayed":30}}`),
		Entry("notInTheLast", NotInTheLast{"lastPlayed": 30.0}, `{"notInTheLast":{"lastPlayed":30}}`),
		Entry("inPlaylist", InPlaylist{"id": "deadbeef-dead-beef"}, `{"inPlaylist":{"id":"deadbeef-dead-beef"}}`),
		Entry("notInPlaylist", NotInPlaylist{"id": "deadbeef-dead-beef"}, `{"notInPlaylist":{"id":"deadbeef-dead-beef"}}`),
	)
})
