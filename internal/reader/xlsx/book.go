// Package xlsx streams rows from Office Open XML workbooks.
package xlsx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"

	"gridview/internal/reader/excel"
	"gridview/internal/table"
)

// Book is an opened workbook. Sheets are read on demand.
type Book struct {
	files   map[string]*zip.File
	names   []string
	paths   []string
	strings []string
	formats []excel.Format
	dates   excel.DateSystem
}

type rel struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

type relsFile struct {
	Rels []rel `xml:"Relationship"`
}

type workbookFile struct {
	Pr struct {
		Date1904   string `xml:"date1904,attr"`
		DateCompat string `xml:"dateCompatibility,attr"`
	} `xml:"workbookPr"`
	Sheets []struct {
		Name string `xml:"name,attr"`
		ID   string `xml:"id,attr"`
	} `xml:"sheets>sheet"`
}

type stylesFile struct {
	NumFmts []struct {
		ID   int    `xml:"numFmtId,attr"`
		Code string `xml:"formatCode,attr"`
	} `xml:"numFmts>numFmt"`
	Xfs []struct {
		NumFmtID int `xml:"numFmtId,attr"`
	} `xml:"cellXfs>xf"`
}

// Open reads the workbook structure, shared strings and styles.
func Open(r io.ReaderAt, size int64) (*Book, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, err
	}
	b := &Book{files: make(map[string]*zip.File, len(zr.File))}
	for _, f := range zr.File {
		b.files[strings.ToLower(f.Name)] = f
	}
	var root relsFile
	if err := b.decode("_rels/.rels", &root); err != nil {
		return nil, err
	}
	wbPath := target("", find(root.Rels, "/officeDocument"), "xl/workbook.xml")
	if _, ok := b.files[strings.ToLower(wbPath)]; !ok {
		return nil, fmt.Errorf("not an xlsx workbook")
	}
	dir := path.Dir(wbPath)
	var wb workbookFile
	if err := b.decode(wbPath, &wb); err != nil {
		return nil, err
	}
	var rels relsFile
	if err := b.decode(path.Join(dir, "_rels", path.Base(wbPath)+".rels"), &rels); err != nil {
		return nil, err
	}
	b.dates = dateSystem(wb.Pr.Date1904, wb.Pr.DateCompat)
	for _, s := range wb.Sheets {
		for _, r := range rels.Rels {
			if r.ID == s.ID {
				b.names = append(b.names, s.Name)
				b.paths = append(b.paths, target(dir, r.Target, ""))
			}
		}
	}
	if p := target(dir, find(rels.Rels, "/sharedStrings"), ""); p != "" {
		if err := b.readStrings(p); err != nil {
			return nil, err
		}
	}
	if p := target(dir, find(rels.Rels, "/styles"), ""); p != "" {
		if err := b.readStyles(p); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// Sheets returns the sheet names in workbook order.
func (b *Book) Sheets() []string {
	return b.names
}

// Rows streams a sheet chosen by name or 1-based index; empty picks the first.
func (b *Book) Rows(sheet string) (table.Rows, error) {
	i, err := excel.PickSheet(b.names, sheet)
	if err != nil {
		return nil, err
	}
	f, ok := b.files[strings.ToLower(b.paths[i])]
	if !ok {
		return nil, fmt.Errorf("sheet %q: missing part %s", b.names[i], b.paths[i])
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	return newRows(b, newPrefetch(rc, 256<<10, 3))
}

func (b *Book) decode(name string, v any) error {
	f, ok := b.files[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("missing part %s", name)
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return xml.NewDecoder(rc).Decode(v)
}

func (b *Book) readStyles(name string) error {
	var s stylesFile
	if err := b.decode(name, &s); err != nil {
		return err
	}
	custom := make(map[int]string, len(s.NumFmts))
	for _, f := range s.NumFmts {
		custom[f.ID] = f.Code
	}
	b.formats = make([]excel.Format, len(s.Xfs))
	for i, xf := range s.Xfs {
		if code, ok := custom[xf.NumFmtID]; ok {
			b.formats[i] = excel.ParseFormat(code)
		} else {
			b.formats[i] = excel.Builtin(xf.NumFmtID)
		}
	}
	return nil
}

func (b *Book) readStrings(name string) error {
	f, ok := b.files[strings.ToLower(name)]
	if !ok {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	pf := newPrefetch(rc, 256<<10, 3)
	defer pf.Close()
	sc := newScanner(pf, scanSize)
	var cur []byte
	inText, skip := false, 0
	for {
		text, err := sc.text()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if inText && skip == 0 {
			cur = unescape(cur, text)
		}
		tag, err := sc.tag()
		if err != nil {
			return err
		}
		name, kind := parseTag(tag)
		if kind == otherTag {
			if c, ok := cdata(tag); ok && inText && skip == 0 {
				cur = append(cur, c...)
			}
			continue
		}
		switch string(name) {
		case "si":
			switch kind {
			case startTag:
				cur = cur[:0]
			case endTag:
				b.strings = append(b.strings, string(cur))
			case emptyTag:
				b.strings = append(b.strings, "")
			}
		case "t":
			inText = kind == startTag
		case "rPh":
			if kind == startTag {
				skip++
			} else if kind == endTag {
				skip--
			}
		}
	}
}

func dateSystem(date1904, compat string) excel.DateSystem {
	switch {
	case date1904 == "1" || date1904 == "true":
		return excel.Date1904
	case compat == "0" || compat == "false":
		return excel.Date1899
	}
	return excel.Date1900
}

func find(rels []rel, typeSuffix string) string {
	for _, r := range rels {
		if strings.HasSuffix(r.Type, typeSuffix) {
			return r.Target
		}
	}
	return ""
}

func target(dir, t, fallback string) string {
	switch {
	case t == "":
		return fallback
	case strings.HasPrefix(t, "/"):
		return t[1:]
	}
	return path.Join(dir, t)
}
