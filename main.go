package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const placeHolderUrl = "https://v2.jokeapi.dev/joke/Any"

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("An error occured: %v", err)
		os.Exit(1)
	}
}
