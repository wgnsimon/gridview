# Builds the Excel test fixtures in internal/reader/testdata with a local Excel.
# Usage: powershell -ExecutionPolicy Bypass -File tools/fixtures/excel.ps1
param([string]$Out = (Join-Path $PSScriptRoot '..\..\internal\reader\testdata'))

$ErrorActionPreference = 'Stop'
# COM calls carry the caller's locale; en-US keeps number format codes English.
function F($cell, [string]$fmt) { [void]$cell.GetType().InvokeMember('NumberFormat', [Reflection.BindingFlags]::SetProperty, $null, $cell, @($fmt), [Globalization.CultureInfo]'en-US') }
$Out = (Resolve-Path $Out).Path
$m = [Type]::Missing

function U([int[]]$codes) { -join ($codes | ForEach-Object { [char]::ConvertFromUtf32($_) }) }

$xl = New-Object -ComObject Excel.Application
$xl.Visible = $false
$xl.DisplayAlerts = $false
try {
    $wb = $xl.Workbooks.Add()
    while ($wb.Worksheets.Count -lt 4) { [void]$wb.Worksheets.Add($m, $wb.Worksheets.Item($wb.Worksheets.Count)) }

    $s = $wb.Worksheets.Item(1)
    $s.Name = 'Data'
    $h = 'Text', 'Number', 'Date', 'Time', 'DateTime', 'Percent', 'Bool', 'Formula', 'Error', 'Empty', 'Unicode'
    for ($i = 0; $i -lt $h.Count; $i++) { $s.Cells.Item(1, ($i + 1)).Value2 = $h[$i] }

    $s.Cells.Item(2, 1).Value2 = 'Anna'
    $s.Cells.Item(2, 2).Value2 = 42
    $c = $s.Cells.Item(2, 3); $c.Value2 = 45322; F $c 'm/d/yyyy'
    $c = $s.Cells.Item(2, 4); $c.Value2 = 0.5729166666666666; F $c 'h:mm:ss'
    $c = $s.Cells.Item(2, 5); $c.Value2 = 45322.5729166666666; F $c 'm/d/yyyy h:mm'
    $c = $s.Cells.Item(2, 6); $c.Value2 = 0.25; F $c '0%'
    $s.Cells.Item(2, 7).Value2 = $true
    $s.Cells.Item(2, 8).Formula = '=B2*2'
    $s.Cells.Item(2, 9).Formula = '=1/0'
    $s.Cells.Item(2, 11).Value2 = 'M' + (U 0xFC) + 'ller ' + (U 0x2615) + ' ' + (U 0x7530, 0x4E2D)

    $s.Cells.Item(3, 1).Value2 = 'Bo, "quoted"'
    $s.Cells.Item(3, 2).Value2 = -3.5
    $c = $s.Cells.Item(3, 3); $c.Value2 = 61; F $c 'dd.mm.yyyy'
    $c = $s.Cells.Item(3, 6); $c.Value2 = 0.125; F $c '0.00%'
    $s.Cells.Item(3, 7).Value2 = $false
    $s.Cells.Item(3, 8).Formula = '=0.1+0.2'
    $s.Cells.Item(3, 9).Formula = '=NA()'
    $s.Cells.Item(3, 11).Value2 = 'Emoji ' + (U 0x1F600)

    $s.Cells.Item(4, 1).Value2 = "two`nlines"
    $c = $s.Cells.Item(4, 2); $c.Value2 = 1234567.891; F $c '0.00'
    $c = $s.Cells.Item(4, 3); $c.Value2 = 60; F $c 'm/d/yyyy'
    $s.Cells.Item(4, 8).Formula = '=A2&"-"&A3'

    $c = $s.Cells.Item(6, 1); F $c '@'; $c.Value2 = '00123'
    $s.Cells.Item(6, 11).Value2 = 'End'

    $s = $wb.Worksheets.Item(2)
    $s.Name = 'Second sheet'
    $s.Cells.Item(1, 1).Value2 = 'x'
    $s.Cells.Item(1, 2).Value2 = 'y'
    $s.Cells.Item(2, 1).Value2 = 1
    $s.Cells.Item(2, 2).Formula = '=A2+1'
    $s.Cells.Item(3, 1).Value2 = 2.5
    $s.Cells.Item(3, 2).Formula = '=A3*2'

    $wb.Worksheets.Item(3).Name = 'Empty'

    $s = $wb.Worksheets.Item(4)
    $s.Name = 'Many'
    $s.Cells.Item(1, 1).Value2 = 'n'
    $s.Cells.Item(1, 2).Value2 = 'text'
    for ($i = 1; $i -le 1200; $i++) {
        $t = ('Row {0:D4} ' -f $i) + (U 0xE4, 0xF6, 0xFC) + ' ' + (U 0xDF)
        if ($i % 2 -eq 1) { $t += ' ' + (U 0x20AC) + (U 0x141) }
        $r = $i + 1
        $s.Range("A$r").Value2 = [double]$i
        $s.Range("B$r").Value2 = [string]$t
    }

    $wb.Worksheets.Item(1).Activate()
    $wb.SaveAs((Join-Path $Out 'types.xlsx'), 51)
    $wb.SaveAs((Join-Path $Out 'types.xls'), 56)
    $wb.SaveAs((Join-Path $Out 'excel-local.csv'), 6, $m, $m, $m, $m, $m, $m, $m, $m, $m, $true)
    $wb.SaveAs((Join-Path $Out 'excel-utf8.csv'), 62, $m, $m, $m, $m, $m, $m, $m, $m, $m, $true)
    $wb.SaveAs((Join-Path $Out 'excel-unicode.txt'), 42)
    $wb.Close($false)
}
finally {
    $xl.Quit()
    [void][Runtime.InteropServices.Marshal]::ReleaseComObject($xl)
}

# LibreOffice writes both formats differently, so its re-saves are fixtures too.
$soffice = Join-Path $env:ProgramFiles 'LibreOffice\program\soffice.exe'
if (Test-Path $soffice) {
    $tmp = Join-Path ([IO.Path]::GetTempPath()) ('gridview-lo-' + [guid]::NewGuid())
    $profile = 'file:///' + ($tmp -replace '\\', '/') + '/profile'
    & $soffice "-env:UserInstallation=$profile" --headless --convert-to xls --outdir $tmp (Join-Path $Out 'types.xlsx') | Out-Null
    & $soffice "-env:UserInstallation=$profile" --headless --convert-to 'xlsx:Calc MS Excel 2007 XML' --outdir $tmp (Join-Path $Out 'types.xls') | Out-Null
    Copy-Item (Join-Path $tmp 'types.xls') (Join-Path $Out 'libreoffice.xls')
    Copy-Item (Join-Path $tmp 'types.xlsx') (Join-Path $Out 'libreoffice.xlsx')
    Remove-Item -Recurse -Force $tmp
}
