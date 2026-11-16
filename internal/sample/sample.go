// Package sample generates deterministic order data as CSV or xlsx for tests and large sample files.
package sample

import (
	"archive/zip"
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"math/rand/v2"
	"strconv"
	"strings"
)

var (
	firstNames = []string{"Anna", "Jörg", "Peter", "Lena", "Ömer", "Zoë", "Max", "Sophie", "Liam", "Mia", "Lukas", "Emma", "Jonas", "Hannah", "Felix", "Lea"}
	lastNames  = []string{"Müller", "Schmidt", "Schneider", "Fischer", "Weber", "Meyer", "Wagner", "Becker", "Groß", "Weiß", "Yılmaz", "Chen", "Bauer", "Koch"}
	cities     = []string{"Berlin", "Hamburg", "München", "Köln", "Frankfurt am Main", "Stuttgart", "Düsseldorf", "Leipzig", "Dresden", "Nürnberg"}
	products   = []string{"Laptop", "Monitor 27\"", "Tastatur", "Maus", "USB-C Hub", "Headset", "Webcam", "Dockingstation", "SSD 1TB", "Drucker, Laser"}
	states     = []string{"offen", "bezahlt", "versendet", "storniert", "retour"}
	notes      = []string{"", "", "", "", "Express", "Geschenkverpackung", "Bitte beim Nachbarn abgeben", "Rechnung an Firma, nicht privat", "Lieferung \"dringend\""}
)

// Header is the first row of every generated table.
var Header = []string{"bestell_id", "datum", "kunde", "stadt", "produkt", "menge", "einzelpreis", "gesamt", "status", "notiz"}

type order struct {
	id, day, qty            int
	customer, city, product string
	price                   float64
	status, note            string
}

func orders(n int, fn func(o order)) {
	r := rand.New(rand.NewPCG(42, 7))
	for i := range n {
		o := order{
			id:       100001 + i,
			day:      43831 + r.IntN(6*365),
			qty:      1 + r.IntN(10),
			customer: firstNames[r.IntN(len(firstNames))] + " " + lastNames[r.IntN(len(lastNames))],
			city:     cities[r.IntN(len(cities))],
			product:  products[r.IntN(len(products))],
			price:    float64(499+r.IntN(200000)) / 100,
			status:   states[r.IntN(len(states))],
			note:     notes[r.IntN(len(notes))],
		}
		fn(o)
	}
}

// Rows returns n generated rows as strings, as the readers would show them.
func Rows(n int) [][]string {
	var rows [][]string
	orders(n, func(o order) {
		rows = append(rows, o.strings())
	})
	return rows
}

func (o order) strings() []string {
	return []string{
		strconv.Itoa(o.id), date(o.day), o.customer, o.city, o.product, strconv.Itoa(o.qty),
		strconv.FormatFloat(o.price, 'f', 2, 64), strconv.FormatFloat(o.price*float64(o.qty), 'f', 2, 64), o.status, o.note,
	}
}

func date(serial int) string {
	y, m, d := 1899, 12, 30+serial
	for {
		days := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}[m-1]
		if m == 2 && y%4 == 0 && (y%100 != 0 || y%400 == 0) {
			days = 29
		}
		if d <= days {
			return fmt.Sprintf("%04d-%02d-%02d", y, m, d)
		}
		d -= days
		if m++; m > 12 {
			m, y = 1, y+1
		}
	}
}

// WriteCSV writes a header and n rows.
func WriteCSV(w io.Writer, n int) error {
	bw := bufio.NewWriterSize(w, 1<<20)
	cw := csv.NewWriter(bw)
	cw.Write(Header)
	orders(n, func(o order) {
		cw.Write(o.strings())
	})
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	return bw.Flush()
}

// WriteXLSX writes a workbook with one sheet holding a header and n rows.
func WriteXLSX(w io.Writer, n int) error {
	zw := zip.NewWriter(w)
	strs := map[string]int{}
	var list []string
	index := func(s string) int {
		i, ok := strs[s]
		if !ok {
			i = len(list)
			strs[s] = i
			list = append(list, s)
		}
		return i
	}
	sheet, err := zw.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return err
	}
	sw := bufio.NewWriterSize(sheet, 1<<20)
	sw.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	sw.WriteString(`<row r="1">`)
	for c, h := range Header {
		fmt.Fprintf(sw, `<c r="%s1" t="s"><v>%d</v></c>`, col(c), index(h))
	}
	sw.WriteString(`</row>`)
	row := 2
	orders(n, func(o order) {
		fmt.Fprintf(sw, `<row r="%d">`, row)
		cell := func(c int, attrs, v string) {
			fmt.Fprintf(sw, `<c r="%s%d"%s><v>%s</v></c>`, col(c), row, attrs, v)
		}
		text := func(c int, s string) {
			if s != "" {
				cell(c, ` t="s"`, strconv.Itoa(index(s)))
			}
		}
		cell(0, "", strconv.Itoa(o.id))
		cell(1, ` s="1"`, strconv.Itoa(o.day))
		text(2, o.customer)
		text(3, o.city)
		text(4, o.product)
		cell(5, "", strconv.Itoa(o.qty))
		cell(6, ` s="2"`, strconv.FormatFloat(o.price, 'f', -1, 64))
		cell(7, ` s="2"`, strconv.FormatFloat(o.price*float64(o.qty), 'f', -1, 64))
		text(8, o.status)
		text(9, o.note)
		sw.WriteString(`</row>`)
		row++
	})
	sw.WriteString(`</sheetData></worksheet>`)
	if err := sw.Flush(); err != nil {
		return err
	}
	var ss strings.Builder
	fmt.Fprintf(&ss, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`, len(list), len(list))
	for _, s := range list {
		ss.WriteString(`<si><t>`)
		escape(&ss, s)
		ss.WriteString(`</t></si>`)
	}
	ss.WriteString(`</sst>`)
	parts := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Bestellungen" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`,
		"xl/styles.xml":              `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><cellXfs count="3"><xf numFmtId="0"/><xf numFmtId="14" applyNumberFormat="1"/><xf numFmtId="2" applyNumberFormat="1"/></cellXfs></styleSheet>`,
		"xl/sharedStrings.xml":       ss.String(),
	}
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", "xl/styles.xml", "xl/sharedStrings.xml"} {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(f, parts[name]); err != nil {
			return err
		}
	}
	return zw.Close()
}

func col(c int) string {
	return string(rune('A' + c))
}

func escape(b *strings.Builder, s string) {
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
}
