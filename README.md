# 🌒  PosTUI

![GitHub](https://img.shields.io/github/license/bramca/postui)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/bramca/postui)

![postui](.img/postui.png)

Terminal User Interface for testing API requests written in [Golang](https://go.dev/).

Using the [bubbletea](https://github.com/charmbracelet/bubbletea) framework.

## 🌓 Installation

`go install github.com/bramca/postui/cmd/postui@latest`
or check out the [releases](https://github.com/bramca/postui/releases)

## 🌔 Features

- Never leave your keyboard with *vim* like navigation and keybindings
- Save your requests in a collection *json* file
- Edit your collection *json* in **edit** mode
- Quickstart your collection by providing an OpenAPI specification
- Set headers using *env* variables with bracket expansion (e.g. {{ENV_VAR}})
- Parse / transform response output with *jq* query
- Copy current requests *curl* command

## 🌕 Usage

`postui -f <collection.json>`

or

`postui -s <openapi-spec.yaml> -v <spec version>`

### Options
- `-specfile, -s [optional]`
    * path to your openapi specification file
<br><br>
- `-specversion, -v [optional]`
    * specify the major version of your spec
    * values: 2, 3
<br><br>
- `-collectionfile, -f [optional]`
    * path to api collection file

### UI Components

The TUI consists of two main **views**:

- The **Top** view with inputs such as request url, request method and optional *jq* query

- The **Bottom** view with multiple tabs: `Collection`, `Request Headers`, `Request Body`, `Response Body`, `Response Headers`

### Keybindings
help:
- `ctrl+h`
    * show all keybindings

view navigation:
- `tab`
    * next view (go to top or bottom view)
- `shift+tab`
    * previous view (go to top or bottom view)
- `alt+]`
    * next tab in view
- `alt+[`
    * previous tab in view
- `up/k`
    * go up
- `down/j`
    * go down
- `h`
    * scroll left
    * go back in collection list selection
- `l`
    * scroll right
    * select item in collection list
- `ctrl+b`
    * scroll up multiple lines
- `ctrl+f`
    * scroll down multiple lines
- `g`
    * scroll to the top
- `G`
    * scroll to the bottom
- `ctrl+k`
    * move cursor up in edit mode
- `ctrl+j`
    * move cursor down in edit mode

actions:
- `ctrl+r`
    * run current API request
- `alt+a`
    * add current request to collection
- `alt+x`
    * copy view content to clipboard
- `ctrl+v`
    * clipboard paste
- `ctrl+s`
    * save current collection
- `alt+c`
    * copy current requests curl command
- `ctrl+c`
    * quit application
- `ctrl+l`
    * toggle collection edit mode
- `alt+l`
    * focus on collection view
