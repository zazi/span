package jats

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kapsteur/franco"
	"github.com/miku/span/container"
	"github.com/miku/span/finc"
	"golang.org/x/text/language"
)

var (
	errNoDOI          = errors.New("DOI is missing")
	errNotImplemented = errors.New("not implemented")
)

var (
	// Restricts the possible languages for detection.
	acceptedLanguages = container.NewStringSet("deu", "eng", "fra", "ita", "spa")

	// Candidate patterns for parsing publishing dates.
	datePatterns = []string{
		"2006",
		"2006-",
		"2006-1",
		"2006-01",
		"2006-1-2",
		"2006-1-02",
		"2006-01-2",
		"2006-01-02",
		"2006-Jan",
		"2006-January",
		"2006-Jan-2",
		"2006-Jan-02",
		"2006-January-2",
		"2006-January-02",
		"2006-x",
		"2006-xx",
		"2006-x-x",
		"2006-x-xx",
		"2006-xx-x",
		"2006-xx-xx",
	}
)

// PubDate represents a publication date. Typical type values are ppub and epub.
type PubDate struct {
	Type  string `xml:"pub-type,attr"`
	Month struct {
		XMLName xml.Name `xml:"month"`
		Value   string   `xml:",chardata"`
	}
	Year struct {
		XMLName xml.Name `xml:"year"`
		Value   string   `xml:",chardata"`
	}
	Day struct {
		XMLName xml.Name `xml:"day"`
		Value   string   `xml:",chardata"`
	}
}

// Article mirrors a JATS article element.
type Article struct {
	XMLName xml.Name `xml:"article"`
	Front   struct {
		XMLName xml.Name `xml:"front"`
		Journal struct {
			ID struct {
				XMLName xml.Name `xml:"journal-id"`
				Type    string   `xml:"journal-id-type,attr"`
				Value   string   `xml:",chardata"`
			}
			ISSN []struct {
				Type  string `xml:"pub-type,attr"`
				Value string `xml:",chardata"`
			} `xml:"issn"`
			TitleGroup struct {
				XMLName      xml.Name `xml:"journal-title-group"`
				JournalTitle struct {
					XMLName xml.Name `xml:"journal-title"`
					Title   string   `xml:",chardata"`
				}
				AbbreviatedTitle struct {
					XMLName xml.Name `xml:"abbrev-journal-title"`
					Title   string   `xml:",chardata"`
					Type    string   `xml:"abbrev-type,attr"`
				}
			}
			Publisher struct {
				XMLName xml.Name `xml:"publisher"`
				Name    struct {
					XMLName xml.Name `xml:"publisher-name"`
					Value   string   `xml:",chardata"`
				}
			}
		} `xml:"journal-meta"`
		Article struct {
			XMLName xml.Name `xml:"article-meta"`
			ID      []struct {
				Type  string `xml:"pub-id-type,attr"`
				Value string `xml:",chardata"`
			} `xml:"article-id"`
			TitleGroup struct {
				XMLName xml.Name `xml:"title-group"`
				Title   struct {
					XMLName xml.Name `xml:"article-title"`
					Value   string   `xml:",chardata"`
				}
				Subtitle struct {
					XMLName xml.Name `xml:"subtitle"`
					Value   string   `xml:",chardata"`
				}
			}
			ContribGroup struct {
				XMLName xml.Name `xml:"contrib-group"`
				Contrib []struct {
					Type      string `xml:"contrib-type,attr"`
					XLinkType string `xml:"xlink.type,attr"`
					Name      struct {
						XMLName xml.Name `xml:"name"`
						Style   string   `xml:"name-style"`
						Surname struct {
							XMLName xml.Name `xml:"surname"`
							Value   string   `xml:",chardata"`
						}
						GivenNames struct {
							XMLName xml.Name `xml:"given-names"`
							Value   string   `xml:",chardata"`
						}
					}
					StringName struct {
						XMLName xml.Name `xml:"string-name"`
						Surname struct {
							XMLName xml.Name `xml:"surname"`
							Value   string   `xml:",chardata"`
						}
						GivenNames struct {
							XMLName xml.Name `xml:"given-names"`
							Value   string   `xml:",chardata"`
						}
						X struct {
							XMLName xml.Name `xml:"x"`
							Value   string   `xml:",chardata"`
						}
						Suffix struct {
							XMLName xml.Name `xml:"suffix"`
							Value   string   `xml:",chardata"`
						}
					}
				} `xml:"contrib"`
			}
			Categories struct {
				XMLName       xml.Name `xml:"article-categories"`
				SubjectGroups []struct {
					Type     string `xml:"subj-group-type,attr"`
					Subjects []struct {
						Value string `xml:",chardata"`
					} `xml:"subject"`
				} `xml:"subj-group"`
			} `xml:"article-categories"`
			PubDates []PubDate `xml:"pub-date"`
			Volume   struct {
				XMLName xml.Name `xml:"volume"`
				Value   string   `xml:",chardata"`
			}
			Issue struct {
				XMLName xml.Name `xml:"issue"`
				Value   string   `xml:",chardata"`
			}
			FirstPage struct {
				XMLName xml.Name `xml:"fpage"`
				Value   string   `xml:",chardata"`
			}
			LastPage struct {
				XMLName xml.Name `xml:"lpage"`
				Value   string   `xml:",chardata"`
			}
			Permissions struct {
				XMLName       xml.Name `xml:"permissions"`
				CopyrightYear struct {
					XMLName xml.Name `xml:"copyright-year"`
					Value   string   `xml:",chardata"`
				}
				CopyrightStatement struct {
					XMLName xml.Name `xml:"copyright-statement"`
					Value   string   `xml:",chardata"`
				}
			}
			Abstract struct {
				XMLName xml.Name `xml:"abstract"`
				Value   string   `xml:",innerxml"`
				Lang    string   `xml:"lang,attr"`
			}
			TranslatedAbstract struct {
				XMLName xml.Name `xml:"trans-abstract"`
				Lang    string   `xml:"lang,attr"`
				Title   struct {
					XMLName xml.Name `xml:"title"`
					Value   string   `xml:",innerxml"`
				}
			}
			KeywordGroup struct {
				XMLName xml.Name `xml:"kwd-group"`
				Title   struct {
					XMLName xml.Name `xml:"title"`
					Value   string   `xml:",chardata"`
				}
				Keywords []struct {
					XMLName xml.Name `xml:"kwd"`
					Value   string   `xml:",chardata"`
				}
			}
			SelfURI struct {
				XMLName xml.Name `xml:"self-uri"`
				Value   string   `xml:"href,attr"`
			}
		}
	}
	Body struct {
		XMLName xml.Name `xml:"body"`
		Section struct {
			XMLName xml.Name `xml:"sec"`
			Type    string   `xml:"sec-type,attr"`
			Value   string   `xml:",innerxml"`
		}
	}
}

// DOI is a convenience shortcut to get the DOI.
// It is an error, if there is no DOI.
func (article *Article) DOI() (s string, err error) {
	for _, id := range article.Front.Article.ID {
		if id.Type == "doi" {
			return id.Value, nil
		}
	}
	return s, errNoDOI
}

// identifiers is a helper struct.
type Identifiers struct {
	DOI      string
	URL      string
	RecordID string
}

// identifiers returns the doi and the dependent url and recordID in a struct.
// It is an error, if there is no DOI.
func (article *Article) Identifiers() (Identifiers, error) {
	return Identifiers{}, errNotImplemented
}

// Authors returns the authors as slice.
// TODO(miku): get rid of cross-format dependency.
func (article *Article) Authors() []finc.Author {
	var authors []finc.Author
	group := article.Front.Article.ContribGroup
	for _, contrib := range group.Contrib {
		if contrib.Type != "author" {
			continue
		}
		authors = append(authors, finc.Author{
			LastName:  contrib.Name.Surname.Value,
			FirstName: contrib.Name.GivenNames.Value})
	}
	return authors
}

// CombinedTitle returns a longish title.
func (article *Article) CombinedTitle() string {
	group := article.Front.Article.TitleGroup
	if group.Title.Value != "" {
		if group.Subtitle.Value != "" {
			return strings.TrimSpace(fmt.Sprintf("%s : %s", group.Title.Value, group.Subtitle.Value))
		}
		return strings.TrimSpace(group.Title.Value)
	}
	if group.Subtitle.Value != "" {
		return strings.TrimSpace(group.Subtitle.Value)
	}
	return ""
}

// ISSN returns a list of ISSNs associated with this article.
func (article *Article) ISSN() (issns []string) {
	for _, issn := range article.Front.Journal.ISSN {
		issns = append(issns, issn.Value)
	}
	return
}

// Headings returns heading categories.
func (article *Article) Headings() (hs []string) {
	for _, g := range article.Front.Article.Categories.SubjectGroups {
		if g.Type != "heading" {
			continue
		}
		for _, s := range g.Subjects {
			hs = append(hs, s.Value)
		}
	}
	return
}

// Subjects returns subjects, that are not headings.
func (article *Article) Subjects() (ss []string) {
	for _, g := range article.Front.Article.Categories.SubjectGroups {
		if g.Type == "heading" {
			continue
		}
		for _, s := range g.Subjects {
			ss = append(ss, s.Value)
		}
	}
	return
}

// PageCount return the number of pages as string, or an empty string.
func (article *Article) PageCount() (s string) {
	first, err := strconv.Atoi(article.Front.Article.FirstPage.Value)
	if err != nil {
		return
	}
	last, err := strconv.Atoi(article.Front.Article.LastPage.Value)
	if err != nil {
		return
	}
	if last-first > 0 {
		return fmt.Sprintf("%d", last-first)
	}
	return
}

// parsePubDate tries to get a date out of a pubdate.
func (article *Article) parsePubDate(pd PubDate) (t time.Time) {
	var s string
	if pd.Day.Value == "" {
		if pd.Month.Value == "" {
			s = fmt.Sprintf("%s", pd.Year.Value)
		}
		s = fmt.Sprintf("%s-%s", pd.Year.Value, pd.Month.Value)
	} else {
		s = fmt.Sprintf("%s-%s-%s", pd.Year.Value, pd.Month.Value, pd.Day.Value)
	}

	var err error
	for _, p := range datePatterns {
		t, err = time.Parse(p, s)
		if err == nil {
			break
		}
	}
	return t
}

// Date returns this articles' issuing date in a best effort manner.
// Use electronic publication (epub), if available.
func (article *Article) Date() (t time.Time) {
	switch len(article.Front.Article.PubDates) {
	case 0:
		return
	case 1:
		return article.parsePubDate(article.Front.Article.PubDates[0])
	default:
		var index int
		for i, pd := range article.Front.Article.PubDates {
			if pd.Type == "epub" {
				index = i
			}
		}
		return article.parsePubDate(article.Front.Article.PubDates[index])
	}
}

// Languages returns the given and guessed languages
// found in abstract and fulltext. Note: This is slow.
// Skip detection on too short strings.
func (article *Article) Languages() []string {
	set := container.NewStringSet()

	if article.Front.Article.Abstract.Lang != "" {
		base, err := language.ParseBase(article.Front.Article.Abstract.Lang)
		if err == nil {
			set.Add(base.ISO3())
		}
	}

	vals := []string{
		article.Front.Article.Abstract.Value,
		article.Front.Article.TranslatedAbstract.Title.Value,
		article.Body.Section.Value,
	}

	for _, s := range vals {
		if len(s) < 20 {
			continue
		}
		lang := franco.DetectOne(s)
		if !acceptedLanguages.Contains(lang.Code) {
			continue
		}
		if lang.Code == "und" {
			continue
		}
		set.Add(lang.Code)
	}

	return set.Values()
}

// ToInternalSchema converts a jats article into an internal schema.
// This is a basic implementation, different source might implement their own.
func (article *Article) ToIntermediateSchema() (*finc.IntermediateSchema, error) {
	output := finc.NewIntermediateSchema()

	date := article.Date()
	output.RawDate = date.Format("2006-01-02")

	output.Abstract = string(article.Front.Article.Abstract.Value)
	output.ArticleTitle = article.CombinedTitle()
	output.Authors = article.Authors()
	output.Fulltext = article.Body.Section.Value
	output.Genre = "article"
	output.RefType = "JOUR"
	output.Headings = article.Headings()
	output.ISSN = article.ISSN()
	output.Issue = article.Front.Article.Issue.Value

	if article.Front.Journal.TitleGroup.JournalTitle.Title != "" {
		output.JournalTitle = article.Front.Journal.TitleGroup.JournalTitle.Title
	} else {
		output.JournalTitle = article.Front.Journal.TitleGroup.AbbreviatedTitle.Title
	}

	output.Languages = article.Languages()
	output.Publishers = append(output.Publishers, article.Front.Journal.Publisher.Name.Value)
	output.Subjects = article.Subjects()
	output.Volume = article.Front.Article.Volume.Value

	output.StartPage = article.Front.Article.FirstPage.Value
	output.EndPage = article.Front.Article.LastPage.Value
	output.PageCount = article.PageCount()
	output.Pages = fmt.Sprintf("%s-%s", output.StartPage, output.EndPage)

	return output, nil
}
