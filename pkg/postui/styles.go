package postui

import "github.com/charmbracelet/lipgloss"

var (
	highlightColor                  = lipgloss.AdaptiveColor{Light: "#82aaff", Dark: "#B191FF"}
	nonHighlightColor               = lipgloss.AdaptiveColor{Light: "#B5B5B5", Dark: "#535353"}
	responseTimeColor               = lipgloss.AdaptiveColor{Light: "#72acff", Dark: "#c792ea"}
	responseSizeColor               = lipgloss.AdaptiveColor{Light: "#9facf1", Dark: "#a792e2"}
	collectionListTitleColor        = lipgloss.AdaptiveColor{Light: "#1a2c79", Dark: "#4535aa"}
	collectionListActiveColor       = lipgloss.AdaptiveColor{Light: "#527cbc", Dark: "#b05cba"}
	collectionListFilterPromptColor = lipgloss.AdaptiveColor{Light: "#33539e", Dark: "#ed639e"}
	placeHolderColor                = lipgloss.AdaptiveColor{Light: "#B5B5B5", Dark: "#535353"}
	focusedStyle                    = lipgloss.NewStyle().Foreground(highlightColor)
	cursorStyle                     = focusedStyle
	noStyle                         = lipgloss.NewStyle()
	inactiveTabBorder               = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder                 = tabBorderWithBottom("┘", " ", "└")
	inactiveTabStyle                = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(nonHighlightColor)
	activeTabStyle                  = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle                     = lipgloss.NewStyle().BorderForeground(nonHighlightColor).Align(lipgloss.Center).Border(lipgloss.NormalBorder()).UnsetBorderTop()
	spinnerStyle                    = lipgloss.NewStyle().Foreground(highlightColor)
	statusCodeViewStyle             = lipgloss.NewStyle().Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"}).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
	responseTimeViewStyle           = lipgloss.NewStyle().Background(responseTimeColor).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
	responseSizeViewStyle           = lipgloss.NewStyle().Background(responseSizeColor).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
	placeHolderStyle                = lipgloss.NewStyle().Foreground(placeHolderColor)
)

func resetStyles() {
	focusedStyle = lipgloss.NewStyle().Foreground(highlightColor)
	cursorStyle = focusedStyle
	noStyle = lipgloss.NewStyle()
	inactiveTabStyle = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(nonHighlightColor)
	activeTabStyle = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle = lipgloss.NewStyle().BorderForeground(nonHighlightColor).Align(lipgloss.Center).Border(lipgloss.NormalBorder()).UnsetBorderTop()
	spinnerStyle = lipgloss.NewStyle().Foreground(highlightColor)
	statusCodeViewStyle = lipgloss.NewStyle().Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"}).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
	responseTimeViewStyle = lipgloss.NewStyle().Background(responseTimeColor).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
	responseSizeViewStyle = lipgloss.NewStyle().Background(responseSizeColor).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
	placeHolderStyle = lipgloss.NewStyle().Foreground(placeHolderColor)
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right

	return border
}
