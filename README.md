# TeaQLite

[![Test](https://github.com/taigrr/teaqlite/actions/workflows/test.yml/badge.svg)](https://github.com/taigrr/teaqlite/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/taigrr/teaqlite.svg)](https://pkg.go.dev/github.com/taigrr/teaqlite)
[![License: 0BSD](https://img.shields.io/badge/License-0BSD-blue.svg)](LICENSE)

A colorful-but-minimal terminal user interface for browsing SQLite databases built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Table Browser**: Browse all tables in your SQLite database with pagination
- **Fuzzy Search**: Search tables by name using `/` key with ranked results
- **Data Viewer**: View table data with pagination and row highlighting
- **Row Detail Modal**: View individual rows in a 2-column format (Column | Value)
- **Cell Editing**: Edit individual cell values with live database updates
- **SQL Query Interface**: Execute custom SQL queries with parameter support
- **Vim-style Navigation**: `j`/`k` movement, `gg`/`G` jumps, and more
- **Responsive Design**: Adapts to terminal size and fits content to screen

## Install

```bash
go install github.com/taigrr/teaqlite@latest
```

Or build from source:

```bash
git clone https://github.com/taigrr/teaqlite.git
cd teaqlite
go build -o teaqlite .
```

## Usage

```bash
teaqlite <database.db>
```

Example with the included sample database:

```bash
teaqlite sample.db
```

## Keybindings

### Global

| Key      | Action      |
| -------- | ----------- |
| `ctrl+c` | Quit        |
| `ctrl+z` | Suspend     |
| `ctrl+g` | Toggle help |

### Table List

| Key       | Action         |
| --------- | -------------- |
| `↑`/`k`   | Move up        |
| `↓`/`j`   | Move down      |
| `←`/`h`   | Previous page  |
| `→`/`l`   | Next page      |
| `enter`   | Open table     |
| `/`       | Search tables  |
| `gg`      | Jump to start  |
| `G`       | Jump to end    |
| `s`       | SQL query mode |
| `r`       | Refresh        |
| `esc`     | Clear filter   |

### Table Data

| Key       | Action          |
| --------- | --------------- |
| `↑`/`k`   | Move up         |
| `↓`/`j`   | Move down       |
| `←`/`h`   | Previous page   |
| `→`/`l`   | Next page       |
| `enter`   | View row detail |
| `/`       | Search data     |
| `gg`      | Jump to start   |
| `G`       | Jump to end     |
| `q`       | Back to tables  |
| `esc`     | Back to tables  |
| `s`       | SQL query mode  |
| `r`       | Refresh         |

### Row Detail

| Key     | Action    |
| ------- | --------- |
| `↑`/`k` | Move up   |
| `↓`/`j` | Move down |
| `e`     | Edit cell |
| `q`     | Back      |
| `esc`   | Back      |

### SQL Query Mode

| Key     | Action          |
| ------- | --------------- |
| `enter` | Execute query    |
| `esc`   | Back to tables   |
| `↑`/`k` | Navigate results |
| `↓`/`j` | Navigate results |
| `gg`    | Jump to start    |
| `G`     | Jump to end      |
| `e`     | Edit query       |

### Edit Cell

| Key     | Action |
| ------- | ------ |
| `enter` | Save   |
| `esc`   | Cancel |

## License

[0BSD](LICENSE)
