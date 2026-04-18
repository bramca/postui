package main

import (
	"fmt"
	"os"

	"github.com/bramca/postui/pkg/postui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jessevdk/go-flags"
)

var opts struct {
	CollectionFile   string `short:"f" long:"collectionfile" description:"path to a collection json file"`
	CollectionDir    string `short:"d" long:"collectiondir" description:"path to a collection directory"`
	SpecFile         string `short:"s" long:"specfile" description:"path to your openapi specification file" required:"false"`
	SpecMajorVersion int    `short:"v" long:"specversion" choice:"2" choice:"3" description:"specify the major version of your spec" required:"false"`
	SkipTlsVerify    bool   `short:"t" long:"skiptlsverify" required:"false"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Something went wrong with the argument parsing: %v", err)
		os.Exit(1)
	}
	collectionFile := opts.CollectionFile
	collectionDir := opts.CollectionDir
	specFile := opts.SpecFile
	specVersion := opts.SpecMajorVersion
	skipTlsVerify := opts.SkipTlsVerify
	p := tea.NewProgram(postui.InitialModel(collectionDir, collectionFile, specFile, specVersion, skipTlsVerify), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("An error occured: %v", err)
		os.Exit(1)
	}
}
