# gridview

Schneller, leichter Tabellen-Viewer fürs Terminal. Der Befehl heißt `grid` und liest CSV, xlsx und xls ohne Abhängigkeiten außer `golang.org/x/term`.

```bash
go install ./cmd/grid
```

## Bedienung

```bash
grid daten.csv               # Tabelle ausgeben
grid -view daten.xlsx        # interaktive Ansicht
grid -s Umsatz mappe.xls     # Sheet per Name oder 1-basiertem Index
grid -d ";" export.csv       # Trennzeichen erzwingen, sonst automatisch
grid -w 0 daten.csv > out    # nicht auf Terminalbreite kürzen
cat daten.csv | grid         # stdin, auch zusammen mit -view
```

| Flag | Bedeutung |
|---|---|
| `-view`, `-v` | interaktive Ansicht |
| `-sheet`, `-s` | Sheet für xlsx/xls, Name oder Nummer |
| `-d` | CSV-Trennzeichen, `tab` für Tabulator |
| `-width`, `-w` | maximale Zeilenbreite, `0` unbegrenzt, Standard ist die Terminalbreite |

Flags dürfen vor oder nach der Datei stehen.

**Erkannt wird automatisch:** das Format an den ersten Bytes (nicht an der Endung), UTF-8 mit und ohne BOM, UTF-16 (Excel „Unicode-Text“), Windows-1252 (deutsches Excel-CSV), das Trennzeichen `,` `;` Tab `|` sowie Zahlenspalten, die rechtsbündig gesetzt werden.

**Excel-Werte** werden so angezeigt, wie Excel sie meint: Datumswerte als `2024-01-31`, Uhrzeiten als `13:45:00`, Prozente, feste Nachkommastellen, `TRUE`/`FALSE`, Fehler wie `#DIV/0!` und das zwischengespeicherte Ergebnis von Formeln.

### Tasten in `-view`

| Taste | Aktion |
|---|---|
| `↑ ↓` `j k` | Zeile |
| `← →` `h l` | Spalte |
| `PgUp PgDn` `b Leertaste` | Seite |
| `g G` `Home End` | Anfang, Ende |
| `0 $` | erste, letzte Spalte |
| `/` `n` `N` | suchen, nächster, vorheriger Treffer |
| `w` | breite Spalten an/aus |
| `q` `Esc` `Ctrl-C` | beenden |

Große Dateien laden im Hintergrund, die erste Seite erscheint sofort.

## Struktur

```
cmd/grid/                  Flags, Verdrahtung, Ende-zu-Ende- und Golden-Tests
internal/
  table/                   gemeinsame Basis: Rows-Interface, kompakter Zeilenspeicher
  layout/                  gemeinsame Basis: Anzeigebreite, Zellen, Spalten einpassen
  reader/                  Format-Erkennung, CSV, Zeichensätze, Trennzeichen
    excel/                 Zahlenformate, Datumssysteme, Sheet-Auswahl (xlsx und xls)
    xlsx/                  ZIP + streamender XML-Scanner
    xls/                   Compound File + BIFF8
    readertest/            gemeinsame Prüfungen gegen die Fixtures
    testdata/              echte Dateien aus Excel und LibreOffice
  output/                  Textausgabe
  tui/                     interaktive Ansicht
  sample/                  reproduzierbare Testdaten
tools/
  gendata/                 große CSV/xlsx-Dateien erzeugen
  fixtures/excel.ps1       Fixtures mit Excel und LibreOffice neu erzeugen
```

```
cmd/grid → reader, output, tui
output   → layout, table
tui      → layout, table
reader   → table          (xlsx, xls → excel, table)
```

Textausgabe und TUI teilen sich `table` und `layout`, wissen aber nichts voneinander. Die Reader kennen weder Terminal noch Ausgabe.

## Tests

```bash
go test ./...
```

- **Unit-Tests** neben jeder Datei
- **Fixtures** in `internal/reader/testdata`, erzeugt von Excel und LibreOffice: Datumswerte, Formeln, Fehler, Unicode, mehrere Sheets, 1200 gemeinsame Strings über CONTINUE-Records
- **Golden Files** für die komplette Ausgabe aller Fixtures (`cmd/grid/testdata`) und eine TUI-Sitzung (`internal/tui/testdata`). Neu schreiben mit `go test ./... -update`, danach den Diff in git prüfen.
- **Fuzzing**: Der CSV-Parser wird gegen `encoding/csv` verglichen, xlsx und xls gegen kaputte Eingaben

```bash
go test ./internal/reader -run=^$ -fuzz=FuzzCSVMatchesEncodingCSV -fuzztime=60s
```

```bash
go test ./internal/reader/xls -run=^$ -fuzz=FuzzOpen -fuzztime=60s
```
