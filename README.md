<p align="center"><img src="docs/logo.svg" alt="grid" width="246"></p>

`grid` shows CSV and Excel files as a table in the terminal. Just pass the file and `grid` works out the rest by itself: format, character set, delimiter and numeric columns.

```
$ grid customers.csv
┌────┬──────┬────────┐
│ id │ name │ city   │
├────┼──────┼────────┤
│  1 │ Anna │ Berlin │
│  2 │ Ömer │ Köln   │
└────┴──────┴────────┘
```

For large tables there is an interactive view for scrolling and searching.

## Installation

Ready-built binaries for Linux, macOS, Windows and FreeBSD are attached to every [release](https://github.com/wgnsimon/gridview/releases). Download the archive for your system, unpack it and put `grid` (or `grid.exe`) somewhere on your `PATH`.

Every archive carries a build provenance attestation, so you can check that it really came out of this repository's release workflow before you run it. With the [GitHub CLI](https://cli.github.com):

```bash
gh attestation verify grid_v0.1.0_darwin_arm64.tar.gz --repo wgnsimon/gridview
```

`checksums.txt` in the release holds the SHA-256 of every archive.

To build it yourself you need [Go](https://go.dev/dl/) 1.27 or newer.

```bash
git clone https://github.com/wgnsimon/gridview.git
```

```bash
cd gridview
```

```bash
go install -ldflags="-s -w" -trimpath ./cmd/grid
```

`grid` then lives in `$(go env GOPATH)/bin`. If that folder is on your `PATH`, `grid` works in every terminal.

## Getting started

```bash
grid data.csv                # print the table
grid -v data.xlsx            # browse interactively
grid -s Sales book.xls       # open a specific sheet
cat data.csv | grid          # read from stdin
grid -h                      # all options
```

## Printing a table

Without options, `grid` writes the table straight to the terminal, framed with box-drawing lines. The first row is the header and is separated by a rule. Numeric columns are right-aligned.

When the output goes to a file or a pipe, `grid` uses plain ASCII lines instead, so the result is easy to process further:

```
id | name | city
---+------+-------
 1 | Anna | Berlin
 2 | Ömer | Köln
```

**Width.** The table is fitted to the width of the terminal. Cells that are too long are cut with `…`, line breaks inside a cell show as `↵`. If not all columns fit, `grid` says at the end how many are missing:

```
┌─────────┬─────────┬─────────┬─────────┬─────────┬────────┐
│ Text    │  Number │ Date    │ Time    │ DateTi… │ Perce… │
├─────────┼─────────┼─────────┼─────────┼─────────┼────────┤
│ Anna    │      42 │ 2024-0… │ 13:45:… │ 2024-0… │    25% │
│ Bo, "q… │    -3.5 │ 1900-0… │         │         │ 12.50% │
└─────────┴─────────┴─────────┴─────────┴─────────┴────────┘
(5 more columns, use a wider terminal or -view)
```

`-w 120` sets your own width, `-w 0` disables cutting entirely. That is handy when the output goes to a file:

```bash
grid -w 0 book.xlsx > book.txt
```

**Columns.** `-cols` (or `-c`) shows only the columns you list, in that order. Name them by header (case doesn't matter), by number starting at 1, or by range:

```bash
grid -c name,city customers.csv
```

```bash
grid -c 1,4-6,9- wide.xlsx
```

`9-` means column 9 to the last. With `-noheader`, the names `A`, `B`, `C` work too. `-cols` also applies to the interactive view.

**Long tables.** If a table has more than 250 rows, `grid` asks first so the terminal doesn't get flooded:

```
Print 1200 rows? Use -view to browse instead. [y/N]
```

`y` prints the table, anything else cancels. `-y` skips the question. When writing to a file or pipe, `grid` never asks.

## Interactive view

```bash
grid -v data.xlsx
```

The view fills the terminal and keeps the header and its rule pinned at the top. Row numbers are on the left, the current row is highlighted and the current cell within it stands out. The status line at the bottom shows file, row, column and column name. Large files keep loading in the background; the first page appears immediately.

| Key | Action |
|---|---|
| `↑` `↓` or `k` `j` | one row up, down |
| `←` `→` or `h` `l` | one column left, right |
| `PgUp` `PgDn` or `b` `Space` or `Ctrl-B` `Ctrl-F` | one page up, down |
| `Ctrl-U` `Ctrl-D` | half a page up, down |
| `Home` `End` or `gg` `G` | first, last row |
| `H` `M` `L` | top, middle, bottom row on screen |
| `0` `$` | first, last column |
| `/` `?` | search forward, backward |
| `n` `N` | next, previous match |
| `s` `S` | sort by the current column, ascending, descending |
| `&` | show only rows containing a text, empty to show all |
| `Enter` | show the current row as a record |
| `f` | freeze the columns up to the current one, again to unfreeze |
| `w` | toggle full-width columns |
| `q` `Esc` `Ctrl-C` | quit |

**Counts.** Like in vi, a number before a key repeats it: `5j` moves five rows down, `3l` three columns right. `42G` or `42gg` jumps to row 42, `3n` to the third match. `Esc` drops a count you typed.

**Search.** `/` opens an input line at the bottom. Type the search term and press `Enter` to jump. `?` searches backwards, and `n` then keeps going in that direction. Case doesn't matter, and all cells are searched. At the end it wraps around. `Esc` cancels the input.

**Wide columns.** Normally every column is cut to at most 40 characters so more fit side by side. `w` shows them at full width, a second `w` switches back.

**Sorting.** `s` sorts the rows by the current column in ascending order, `S` in descending order. Pressing the same key on the same column again restores the order of the file. Numbers are sorted by value, also with a decimal comma, thousands separators or `%`. Text is sorted ignoring case, with numbers inside it compared by value, so `file2` comes before `file10`. Numbers come before text, empty cells always last. The row numbers on the left keep showing the row in the file, the status line shows the sort column. Sorting works once the file has finished loading.

**Filtering.** `&` opens an input line like the search. After `Enter`, only rows with a cell containing the text are shown, ignoring case; the status line shows the filter. `&` again starts with the current filter so you can change it, an empty filter shows all rows again. Filter and sort combine, and the search then only looks at the filtered rows. Like sorting, filtering works once the file has finished loading.

**Freezing columns.** `f` keeps the columns up to and including the current one on the left while the others scroll, handy for an ID or name column in a wide table. The status line shows how many are frozen; another `f` unfreezes them.

**Record.** `Enter` shows the current row as a list of column names and values, one per line. Long values wrap over several lines and line breaks inside a cell show as real line breaks, so nothing is cut. `↑` `↓` scroll, `←` `→` go to the previous and next row, `Esc`, `Enter` or `q` go back to the table.

The view also works with data from stdin:

```bash
curl -s https://example.com/export.csv | grid -v
```

## File formats

| Format | Notes |
|---|---|
| CSV, TSV, TXT | delimiters `,` `;` tab or `\|` are detected |
| xlsx | Excel 2007 and newer, also from LibreOffice |
| xls | Excel 97 to 2003 |

The format is detected from the file contents, not the extension, so a misnamed file still opens.

### Header row

`grid` takes the first row as the header. Exports often start with a title or a few lines of notes instead; `-skip` drops that many rows from the top, so the real header comes first:

```bash
grid -skip 3 report.csv
```

If the file has no header at all, `-noheader` keeps the first row as data and names the columns `A`, `B`, `C` like a spreadsheet:

```bash
grid -noheader measurements.csv
```

Both work for every format and can be combined. Blank lines in a CSV file don't count, blank rows in a sheet do.

### CSV

Character sets are detected automatically: UTF-8 with and without BOM, UTF-16 (Excel "Unicode Text") and Windows-1252, which Excel writes in many Western European locales when saving as CSV. So accented characters look right even in old exports.

If delimiter detection gets it wrong, set it with `-d`:

```bash
grid -d ";" export.csv
```

```bash
grid -d tab export.txt
```

### Excel

`grid` opens the first sheet. Pick another with `-s`, by name or number (starting at 1). Names are matched case-insensitively.

```bash
grid -s Sales book.xlsx
```

```bash
grid -s 2 book.xlsx
```

If the sheet doesn't exist, the error message lists all available ones.

Cells appear the way Excel displays them:

| In Excel | In grid |
|---|---|
| Date | `2024-01-31` |
| Time | `13:45:00` |
| Date with time | `2024-01-31 13:45:00` |
| Percentage | `25%`, `12.50%` |
| Number with fixed decimals | as many decimals as the format has |
| Boolean | `TRUE`, `FALSE` |
| Error | `#DIV/0!`, `#N/A` |
| Formula | the result Excel last calculated |

`grid` doesn't evaluate formulas itself. It shows the value Excel or LibreOffice stored when the file was last saved.

**Not supported:** encrypted workbooks, xls from Excel 5.0/95 and older, and other formats such as `.ods` or `.xlsb`.

## Options

| Option | Meaning |
|---|---|
| `-view`, `-v` | interactive view |
| `-sheet`, `-s` | sheet for xlsx/xls, name or number starting at 1 |
| `-d` | force the CSV delimiter, `tab` for tab |
| `-width`, `-w` | maximum line width, `0` for unlimited. Defaults to the terminal width |
| `-yes`, `-y` | print long tables without asking |
| `-cols`, `-c` | columns to show: names, numbers or ranges, e.g. `name,3-5` |
| `-skip` | drop this many rows at the top, before the header |
| `-noheader` | the first row is data, columns are named `A`, `B`, `C` |
| `-h` | show help |

Options may come before or after the file name. Without a file name, or with `-`, `grid` reads from stdin.

On error, `grid` writes a message prefixed with `grid:` to stderr and exits with code 1.

## Contributing

Code structure and tests are described in [DEVELOPMENT.md](DEVELOPMENT.md).

## License

[MIT](LICENSE)
