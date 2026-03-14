# 🌒  PosTUI

![GitHub](https://img.shields.io/github/license/bramca/gen-mockserver)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/bramca/gen-mockserver)

![postui](.img/postui.png)

Terminal User Interface for testing API requests written in [Golang](https://go.dev/).

Using the [bubbletea](https://github.com/charmbracelet/bubbletea) framework.

## 🌓 Installation

`go install github.com/bramca/postui/cmd/postui@latest`
or check out the [releases](https://github.com/bramca/postui/releases)

## 🌔 Usage

`postui -f <collection.json>`

or

`postui -s <api-spec.yaml> -v <spec version>`

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

### Keybindings

- `ctrl+h`
    * show all keybindings
- `tab`
    * go to top or bottom view
