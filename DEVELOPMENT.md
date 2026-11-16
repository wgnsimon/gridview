# Development

Notes for anyone working on `grid` itself. The user guide is in the [README](README.md).

## Building

```bash
go install -ldflags="-s -w" -trimpath ./cmd/grid
```

`-s -w` drops the symbol and debug tables, shrinking the binary from 4.2 MB to 2.7 MB. The only dependency is `golang.org/x/term`.

## Structure

```
cmd/grid/                  flags, wiring, end-to-end and golden tests
internal/
  table/                   shared base: Rows interface, compact row store
  layout/                  shared base: display width, cells, fitting columns
  reader/                  format detection, CSV, character sets, delimiters
    excel/                 number formats, date systems, sheet selection (xlsx and xls)
    xlsx/                  ZIP + streaming XML scanner
    xls/                   Compound File + BIFF8
    readertest/            shared checks against the fixtures
    testdata/              real files from Excel and LibreOffice
  output/                  text output
  tui/                     interactive view
  sample/                  reproducible test data
tools/
  gendata/                 generate large CSV/xlsx files
  fixtures/excel.ps1       regenerate the fixtures with Excel and LibreOffice
docs/                      logo
```

```
cmd/grid → reader, output, tui
output   → layout, table
tui      → layout, table
reader   → table          (xlsx, xls → excel, table)
```

Text output and TUI share `table` and `layout` but know nothing about each other. The readers know neither the terminal nor the output.

## Tests

```bash
go test ./...
```

- **Unit tests** next to each file
- **Fixtures** in `internal/reader/testdata`, produced by Excel and LibreOffice: dates, formulas, errors, Unicode, several sheets, 1200 shared strings across CONTINUE records. The CSV exports come from a German Excel on purpose, so they cover `;` delimiters, decimal commas, Windows-1252 and localized values such as `WAHR`.
- **Golden files** for the complete output of a few fixtures (`cmd/grid/testdata`). Rewrite them with `go test ./... -update`, then review the diff in git.
- **Fuzzing**: the CSV parser is compared against `encoding/csv`, xlsx and xls are fed broken input

```bash
go test ./internal/reader -run=^$ -fuzz=FuzzCSVMatchesEncodingCSV -fuzztime=60s
```

```bash
go test ./internal/reader/xls -run=^$ -fuzz=FuzzOpen -fuzztime=60s
```
