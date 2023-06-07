package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"

	. "github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildPaginationLinksAndMeta", func() {
	var (
		totalItems   int32
		params       GetTracksParams
		resourceName string
	)

	BeforeEach(func() {
		totalItems = 150
		resourceName = "api/resource"
	})

	Context("with default page limit and offset", func() {
		BeforeEach(func() {
			l, o := int32(10), int32(0)
			params = GetTracksParams{
				PageLimit:  &l,
				PageOffset: &o,
			}
		})

		It("returns correct pagination links and meta", func() {
			links, meta := buildPaginationLinksAndMeta(totalItems, params, resourceName)

			testLinkEquality(links.First, P("api/resource?page[offset]=0&page[limit]=10"))
			testLinkEquality(links.Last, P("api/resource?page[offset]=140&page[limit]=10"))
			testLinkEquality(links.Next, P("api/resource?page[offset]=10&page[limit]=10"))
			Expect(links.Prev).To(BeNil())

			Expect(meta.CurrentPage).To(Equal(P(int32(1))))
			Expect(meta.TotalItems).To(Equal(P(int32(150))))
			Expect(meta.TotalPages).To(Equal(P(int32(15))))
		})
	})

	Context("with custom page limit and offset", func() {
		BeforeEach(func() {
			params = GetTracksParams{
				PageLimit:  P((PageLimit)(20)),
				PageOffset: P((PageOffset)(40)),
			}
		})

		It("returns correct pagination links and meta", func() {
			links, meta := buildPaginationLinksAndMeta(totalItems, params, resourceName)

			testLinkEquality(links.First, P("api/resource?page[offset]=0&page[limit]=20"))
			testLinkEquality(links.Last, P("api/resource?page[offset]=140&page[limit]=20"))
			testLinkEquality(links.Next, P("api/resource?page[offset]=60&page[limit]=20"))
			testLinkEquality(links.Prev, P("api/resource?page[offset]=20&page[limit]=20"))

			Expect(meta.CurrentPage).To(Equal(P(int32(3))))
			Expect(meta.TotalItems).To(Equal(P(int32(150))))
			Expect(meta.TotalPages).To(Equal(P(int32(8))))
		})
	})

	Context("with various filter params", func() {
		BeforeEach(func() {
			params = GetTracksParams{
				PageLimit:            P((PageLimit)(20)),
				PageOffset:           P((PageOffset)(40)),
				FilterEquals:         &[]string{"property1:value1", "property2:value2"},
				FilterContains:       &[]string{"property3:value3"},
				FilterLessThan:       &[]string{"property4:value4"},
				FilterLessOrEqual:    &[]string{"property5:value5"},
				FilterGreaterThan:    &[]string{"property6:value6"},
				FilterGreaterOrEqual: &[]string{"property7:value7"},
				FilterStartsWith:     &[]string{"property8:value8"},
				FilterEndsWith:       &[]string{"property9:value9"},
			}
		})

		It("returns correct pagination links with filter params", func() {
			links, _ := buildPaginationLinksAndMeta(totalItems, params, resourceName)

			validateLink := func(link *string, expectedOffset string) {
				parsedLink, err := url.Parse(*link)
				Expect(err).NotTo(HaveOccurred())

				queryParams, _ := url.ParseQuery(parsedLink.RawQuery)
				Expect(queryParams["page[offset]"]).To(ConsistOf(expectedOffset))
				Expect(queryParams["page[limit]"]).To(ConsistOf("20"))

				for _, param := range *params.FilterEquals {
					Expect(queryParams["filter[equals]"]).To(ContainElements(param))
				}
				for _, param := range *params.FilterContains {
					Expect(queryParams["filter[contains]"]).To(ContainElement(param))
				}
				for _, param := range *params.FilterLessThan {
					Expect(queryParams["filter[lessThan]"]).To(ContainElement(param))
				}
				for _, param := range *params.FilterLessOrEqual {
					Expect(queryParams["filter[lessOrEqual]"]).To(ContainElement(param))
				}
				for _, param := range *params.FilterGreaterThan {
					Expect(queryParams["filter[greaterThan]"]).To(ContainElement(param))
				}
				for _, param := range *params.FilterGreaterOrEqual {
					Expect(queryParams["filter[greaterOrEqual]"]).To(ContainElement(param))
				}
				for _, param := range *params.FilterStartsWith {
					Expect(queryParams["filter[startsWith]"]).To(ContainElement(param))
				}
				for _, param := range *params.FilterEndsWith {
					Expect(queryParams["filter[endsWith]"]).To(ContainElement(param))
				}
			}

			validateLink(links.First, "0")
			validateLink(links.Last, "140")
			validateLink(links.Next, "60")
			validateLink(links.Prev, "20")
		})
	})
})

var _ = Describe("toSortParams", func() {
	DescribeTable("toSortParams",
		func(sort string, expected string, expectedError error) {
			order, err := toSortParams(&sort)
			Expect(order).To(Equal(expected))
			if expectedError == nil {
				Expect(err).To(BeNil())
			} else {
				Expect(err).To(Equal(expectedError))
			}
		},
		Entry("should handle nil input", "", "", nil),
		Entry("should handle empty input", "", "", nil),
		Entry("should handle single column input", "name", "name asc", nil),
		Entry("should handle single column input with descending order", "-name", "name desc", nil),
		Entry("should handle multiple columns input", "name,,date,", "name asc,date asc", nil),
		Entry("should handle multiple columns input with mixed order and spaces", "name, -age", "name asc,age desc", nil),
		Entry("should handle relationship columns", "-artist.name", "artist.name desc", nil),
		Entry("should return an error for invalid input with invalid prefix", "+name", "", errors.New("invalid sort parameter: +name")),
		Entry("should return an error for invalid prefix in any column", "name,*age", "", errors.New("invalid sort parameter: *age")),
	)
})

var _ = Describe("storeRequestInContext", func() {
	var (
		nextHandler http.Handler
		handler     http.Handler
	)

	BeforeEach(func() {
		nextHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Context().Value(requestInContext).(*http.Request).URL).To(Equal(r.URL))
		})
		handler = storeRequestInContext(nextHandler)
	})

	It("adds the full request object to the context", func() {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
	})
})

func testLinkEquality(link1, link2 *string) {
	parsedLink1, err := url.Parse(*link1)
	Expect(err).NotTo(HaveOccurred())
	queryParams1, _ := url.ParseQuery(parsedLink1.RawQuery)

	parsedLink2, err := url.Parse(*link2)
	Expect(err).NotTo(HaveOccurred())
	queryParams2, _ := url.ParseQuery(parsedLink2.RawQuery)

	Expect(queryParams1).To(Equal(queryParams2))
}
