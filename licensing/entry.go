// Package licensing implements support for KBART and ISIL attachments.
// KBART might contains special fields, that are important in certain contexts.
// Example: "Aargauer Zeitung" could not be associated with a record, because
// there is no ISSN. However, there is a string
// "https://www.wiso-net.de/dosearch?&dbShortcut=AGZ" in the record, which could
// be parsed to yield "AGZ", which could be used to relate a record to this entry
// (e.g. if the record has "AGZ" in a certain field, like x.package).
package licensing

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/miku/span"
	"github.com/miku/span/container"
)

// DateGranularity indicates how complete a date is.
type DateGranularity byte

const (
	GRANULARITY_YEAR DateGranularity = iota
	GRANULARITY_MONTH
	GRANULARITY_DAY
)

var (
	ErrBeforeFirstIssueDate = errors.New("before first issue date")
	ErrAfterLastIssueDate   = errors.New("after last issue date")
	ErrBeforeFirstVolume    = errors.New("before first volume")
	ErrAfterLastVolume      = errors.New("after last volume")
	ErrBeforeFirstIssue     = errors.New("before first issue")
	ErrAfterLastIssue       = errors.New("after last issue")
	ErrInvalidDate          = errors.New("invalid date")

	intPattern = regexp.MustCompile("[0-9]+")
)

// dateFormat groups layout and granularity.
type dateFormat struct {
	layout      string
	granularity DateGranularity
}

// datePatterns are candidate patterns for parsing dates.
var datePatterns = []dateFormat{
	{"2006", GRANULARITY_YEAR},
	{"2006-01-02", GRANULARITY_DAY},
	{"2006-", GRANULARITY_YEAR},
	{"2006-01-2", GRANULARITY_DAY},
	{"2006-01", GRANULARITY_MONTH},
	{"2006-1-02", GRANULARITY_DAY},
	{"2006-1-2", GRANULARITY_DAY},
	{"2006-1", GRANULARITY_MONTH},
	{"2006-Jan-02", GRANULARITY_DAY},
	{"2006-Jan-2", GRANULARITY_DAY},
	{"2006-Jan", GRANULARITY_MONTH},
	{"2006-January-02", GRANULARITY_DAY},
	{"2006-January-2", GRANULARITY_DAY},
	{"2006-January", GRANULARITY_MONTH},
	{"2006-x-x", GRANULARITY_YEAR},
	{"2006-x-xx", GRANULARITY_YEAR},
	{"2006-x", GRANULARITY_YEAR},
	{"2006-xx-x", GRANULARITY_YEAR},
	{"2006-xx-xx", GRANULARITY_YEAR},
	{"2006-xx", GRANULARITY_YEAR},
	{"20060102", GRANULARITY_DAY},
	{"200601", GRANULARITY_MONTH},
}

// Entry contains fields about a licensed or available journal, book, article or
// other resource. First 14 columns are quite stardardized. Further columns may
// contain custom information:
//
// EZB style: own_anchor, package:collection, il_relevance, il_nationwide,
// il_electronic_transmission, il_comment, all_issns, zdb_id
//
// OCLC style: location, title_notes, staff_notes, vendor_id,
// oclc_collection_name, oclc_collection_id, oclc_entry_id, oclc_linkscheme,
// oclc_number, ACTION
//
// See also: http://www.uksg.org/kbart/s5/guidelines/data_field_labels,
// http://www.uksg.org/kbart/s5/guidelines/data_fields
type Entry struct {
	PublicationTitle                   string `csv:"publication_title"`          // "Südost-Forschungen (2014-)", "Theory of Computation"
	PrintIdentifier                    string `csv:"print_identifier"`           // "2029-8692", "9783662479841"
	OnlineIdentifier                   string `csv:"online_identifier"`          // "1533-8606", "9783834960078"
	FirstIssueDate                     string `csv:"date_first_issue_online"`    // "1901", "2008"
	FirstVolume                        string `csv:"num_first_vol_online"`       // "1",
	FirstIssue                         string `csv:"num_first_issue_online"`     // "1"
	LastIssueDate                      string `csv:"date_last_issue_online"`     // "1997", "2008"
	LastVolume                         string `csv:"num_last_vol_online"`        // "25"
	LastIssue                          string `csv:"num_last_issue_online"`      // "1"
	TitleURL                           string `csv:"title_url"`                  // "http://www.karger.com/dne", "http://link.springer.com/10.1007/978-3-658-15644-2"
	FirstAuthor                        string `csv:"first_author"`               // "Borgmann", "Wissenschaftlicher Beirat der Bundesregierung Globale Umweltveränderungen (WBGU)"
	TitleID                            string `csv:"title_id"`                   // "22540", "10.1007/978-3-658-10838-0"
	Embargo                            string `csv:"embargo_info"`               // "P12M", "P1Y", "R20Y"
	CoverageDepth                      string `csv:"coverage_depth"`             // "Volltext", "ebook"
	CoverageNotes                      string `csv:"coverage_notes"`             // ...
	PublisherName                      string `csv:"publisher_name"`             // "via Hein Online", "Springer (formerly: Kluwer)", "DUV"
	OwnAnchor                          string `csv:"own_anchor"`                 // "elsevier_2016_sax", "UNILEIP", "Wiley Custom 2015"
	PackageCollection                  string `csv:"package:collection"`         // "EBSCO:ebsco_bth", "NALAS:natli_aas2", "NALIW:sage_premier"
	InterlibraryRelevance              string `csv:"il_relevance"`               // ...
	InterlibraryNationwide             string `csv:"il_nationwide"`              // ...
	InterlibraryElectronicTransmission string `csv:"il_electronic_transmission"` // "Papierkopie an Endnutzer", "Elektronischer Versand an Endnutzer"
	InterlibraryComment                string `csv:"il_comment"`                 // "Nur im Inland", "il_nationwide"
	AllSerialNumbers                   string `csv:"all_issns"`                  // "1990-0104;1990-0090", "undefined"
	ZDBID                              string `csv:"zdb_id"`                     // "1459367-1" (see also: http://www.zeitschriftendatenbank.de/suche/zdb-katalog.html)
	Location                           string `csv:"location"`                   // ...
	TitleNotes                         string `csv:"title_notes"`                // ...
	StaffNotes                         string `csv:"staff_notes"`                // ...
	VendorID                           string `csv:"vendor_id"`                  // ...
	OCLCCollectionName                 string `csv:"oclc_collection_name"`       // "Springer German Language eBooks 2016 - Full Set", "Wiley Online Library UBCM All Obooks"
	OCLCCollectionID                   string `csv:"oclc_collection_id"`         // "springerlink.de2011fullset", "wiley.ubcmall"
	OCLCEntryID                        string `csv:"oclc_entry_id"`              // "25106066"
	OCLCLinkScheme                     string `csv:"oclc_link_scheme"`           // "wiley.book"
	OCLCNumber                         string `csv:"oclc_number"`                // "122938128"
	Action                             string `csv:"ACTION"`                     // "raw"

	// cache data, that needs to be parsed, for performance
	parsed struct {
		FirstIssueDate time.Time
		LastIssueDate  time.Time
	}
}

// ISSNList returns a list of normalized ISSN from various fields.
func (e *Entry) ISSNList() []string {
	issns := container.NewStringSet()
	for _, issn := range []string{e.PrintIdentifier, e.OnlineIdentifier} {
		s := NormalizeSerialNumber(issn)
		if span.ISSNPattern.MatchString(s) {
			issns.Add(s)
		}
	}
	for _, issn := range FindSerialNumbers(e.AllSerialNumbers) {
		issns.Add(issn)
	}
	return issns.SortedValues()
}

// Covers is a generic method to determine, whether a given date, volume or issue
// is covered by this entry. It takes into account moving walls. If values are not
// defined, we mostly assume they are not constrained.
func (e *Entry) Covers(date, volume, issue string) error {
	t, g, err := parseWithGranularity(date)
	if err != nil {
		return err
	}
	if err := e.containsDateTime(t, g); err != nil {
		return err
	}
	if err := Embargo(e.Embargo).Compatible(t); err != nil {
		return err
	}

	if e.parsed.FirstIssueDate.Year() == t.Year() {
		if e.FirstVolume != "" && volume != "" && findInt(volume) < findInt(e.FirstVolume) {
			return ErrBeforeFirstVolume
		}
		if e.FirstIssue != "" && issue != "" && findInt(issue) < findInt(e.FirstIssue) {
			return ErrBeforeFirstIssue
		}
	}

	if e.parsed.LastIssueDate.Year() == t.Year() {
		if e.LastVolume != "" && volume != "" && findInt(volume) > findInt(e.LastVolume) {
			return ErrAfterLastVolume
		}
		if e.LastIssue != "" && issue != "" && findInt(issue) > findInt(e.LastIssue) {
			return ErrAfterLastIssue
		}
	}
	return nil
}

// begin parses left boundary of license interval, returns a date far in the past
// if it is not defined.
func (e *Entry) begin() time.Time {
	if e.parsed.FirstIssueDate.IsZero() {
		e.parsed.FirstIssueDate = time.Date(1, time.January, 1, 0, 0, 0, 1, time.UTC)
		for _, dfmt := range datePatterns {
			if t, err := time.Parse(dfmt.layout, e.FirstIssueDate); err == nil {
				e.parsed.FirstIssueDate = t
				break
			}
		}
	}
	return e.parsed.FirstIssueDate
}

// beginGranularity returns the begin date with a given granularity.
func (e *Entry) beginGranularity(g DateGranularity) time.Time {
	t := e.begin()
	switch g {
	case GRANULARITY_YEAR:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	case GRANULARITY_MONTH:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return t
	}
}

// end parses right boundary of license interval, returns a date far in the future
// if it is not defined.
func (e *Entry) end() time.Time {
	if e.parsed.LastIssueDate.IsZero() {
		e.parsed.LastIssueDate = time.Date(2364, time.January, 1, 0, 0, 0, 1, time.UTC)
		for _, dfmt := range datePatterns {
			if t, err := time.Parse(dfmt.layout, e.LastIssueDate); err == nil {
				e.parsed.LastIssueDate = t
				break
			}
		}
	}
	return e.parsed.LastIssueDate
}

// endGranularity returns the end date with a given granularity.
func (e *Entry) endGranularity(g DateGranularity) time.Time {
	t := e.end()
	switch g {
	case GRANULARITY_YEAR:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	case GRANULARITY_MONTH:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return t
	}
}

// containsDateTime returns nil, if the given time lies between this entries
// dates. If the given time is the zero value, it will be contained by any
// interval.
func (e *Entry) containsDateTime(t time.Time, g DateGranularity) error {
	if t.IsZero() {
		return nil
	}
	if t.Before(e.beginGranularity(g)) {
		return ErrBeforeFirstIssueDate
	}
	if t.After(e.endGranularity(g)) {
		return ErrAfterLastIssueDate
	}
	return nil
}

// containsDate return nil, if the given date (as string), lies between this
// entries issue dates. The empty string is interpreted as being inside all
// intervals.
func (e *Entry) containsDate(s string) (err error) {
	if s == "" {
		return nil
	}
	t, g, err := parseWithGranularity(s)
	if err != nil {
		return err
	}
	return e.containsDateTime(t, g)
}

// NormalizeSerialNumber tries to transform the input into 1234-567X standard form.
func NormalizeSerialNumber(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)
	if len(s) == 8 {
		return fmt.Sprintf("%s-%s", s[:4], s[4:])
	}
	return s
}

// FindSerialNumbers returns ISSN in standard form in a given string.
func FindSerialNumbers(s string) []string {
	return span.ISSNPattern.FindAllString(s, -1)
}

// parseWithGranularity tries to parse a string into a time. If successful, also
// return the granularity.
func parseWithGranularity(s string) (t time.Time, g DateGranularity, err error) {
	if s == "" {
		return time.Time{}, GRANULARITY_DAY, ErrInvalidDate
	}
	for _, dfmt := range datePatterns {
		t, err = time.Parse(dfmt.layout, s)
		if err != nil {
			continue
		}
		g = getGranularity(dfmt.layout)
		return
	}
	return t, g, ErrInvalidDate
}

// getGranularity returns the granularity for given date layout.
func getGranularity(layout string) DateGranularity {
	for _, dfmt := range datePatterns {
		if dfmt.layout == layout {
			return dfmt.granularity
		}
	}
	return GRANULARITY_DAY
}

// findInt return the first int that is found in s or 0 if there is no number.
func findInt(s string) int {
	if s == "" {
		return 0
	}
	// We expect to see a number most of the time.
	if i, err := strconv.Atoi(s); err == nil {
		return int(i)
	}
	// Otherwise try to parse out a number.
	m := intPattern.FindString(s)
	if m == "" {
		return 0
	}
	i, _ := strconv.ParseInt(m, 10, 32)
	return int(i)
}
