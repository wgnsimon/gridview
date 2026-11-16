# gridview

Ultra-leichter Tabellen-Viewer fürs Terminal, geschrieben in Go. Der Befehl heißt `grid`.

```bash
go run ./cmd/grid input/small.csv
```

```bash
go build -o grid.exe ./cmd/grid
```

Aktuell: CSV als ausgerichtete Tabelle ausgeben. Geplant: xlsx/xls, `-view` (interaktiv), Filter und Sortierung.

## Struktur

```
cmd/grid/main.go          Flags lesen, reader und output verdrahten
internal/
  table/table.go          Rows-Interface: das, worauf sich alle Pakete einigen
  reader/csv.go           Datei → table.Rows
  reader/testdata/        kleine CSVs für die Tests
  output/text.go          table.Rows → ausgerichtete Textausgabe
  output/width.go         Spaltenbreiten berechnen
input/                    große Dateien zum Ausprobieren
```

```
cmd/grid → reader, output
reader   → table
output   → table
table    → nichts
```

## Regeln

1. **`main` kennt kein CSV.** Es ruft nur `reader.Open` und `output.Text.Show` auf.
2. **Zeilen werden gestreamt.** Alles läuft über `table.Rows`, nie über eine komplette `[][]string`.
3. **`output` schreibt auf einen `io.Writer`**, nie direkt auf `os.Stdout`. Tests übergeben einen `bytes.Buffer`.

## Später erweitern

| Erweiterung | Änderung |
|---|---|
| xlsx/xls | `reader/xlsx.go`, `reader/xls.go`; `Open` zieht nach `reader/open.go` und erkennt das Format |
| BOM, Trennzeichen | `reader/detect.go` |
| `-view` | `output/output.go` (Interface), `output/tui/`; `width.go` zieht nach `internal/layout/` |
| Filter, Sortierung | `internal/transform/` |
| End-to-End-Tests | `test/e2e/` |

## Testen

```bash
go test ./...
```
