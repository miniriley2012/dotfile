package server

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/knoebber/dotfile/db"
)

// Tests the contents of all the page templates.
func TestTemplatesHTML(t *testing.T) {
	if err := loadTemplates(); err != nil {
		t.Fatalf("loading templates: %v", err)
	}

	testData := make(map[string]interface{})
	testSession := &db.SessionRecord{}

	for _, template := range pageTemplates.Templates() {
		curr := template.Name()
		if curr[0] == '_' || curr == "pages" {
			// Skip partials and the original.
			continue
		}

		buff := new(bytes.Buffer)
		p := Page{
			Title:        "Test Page",
			Data:         testData,
			Session:      testSession,
			templateName: curr,
		}
		if err := p.writeFromTemplate(buff); err != nil {
			t.Fatalf("failed to write from template: %v", err)
		}

		assertHTMLResponse(t, string(buff.Bytes()), curr)
	}
}

// Asserts that the html body is valid XML and does not have any empty lines.
func assertHTMLResponse(t *testing.T, body, name string) {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.Trim(line, " ") == "" {
			t.Log(body)
			t.Fatalf("%q body: line %d is empty", name, i)
		}
	}

	htmlReader := strings.NewReader(body)
	d := xml.NewDecoder(htmlReader)
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity

	for {
		_, err := d.Token()
		switch err {
		case io.EOF:
			return
		case nil:
		default:
			t.Log(body)
			t.Fatalf("%q body is not valid html: %v", name, err)
		}
	}

}
