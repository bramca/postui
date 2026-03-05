package postui

import "github.com/charmbracelet/lipgloss"

var (
	highlightColor        = lipgloss.AdaptiveColor{Light: "#82aaff", Dark: "#B191FF"}
	focusedStyle          = lipgloss.NewStyle().Foreground(highlightColor)
	cursorStyle           = focusedStyle
	noStyle               = lipgloss.NewStyle()
	inactiveTabBorder     = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder       = tabBorderWithBottom("┘", " ", "└")
	nonHighlightColor     = lipgloss.AdaptiveColor{Light: "#B5B5B5", Dark: "#535353"}
	inactiveTabStyle      = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(nonHighlightColor)
	activeTabStyle        = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle           = lipgloss.NewStyle().BorderForeground(nonHighlightColor).Align(lipgloss.Center).Border(lipgloss.NormalBorder()).UnsetBorderTop()
	spinnerStyle          = lipgloss.NewStyle().Foreground(highlightColor)
	statusCodeViewStyle   = lipgloss.NewStyle().Background(lipgloss.CompleteColor{TrueColor: "#21FF4E"}).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
	responseTimeViewStyle = lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "#72acff", Dark: "#c792ea"}).Foreground(lipgloss.CompleteColor{TrueColor: "#000000"})
	placeHolderStyle      = lipgloss.NewStyle().Foreground(nonHighlightColor)
)

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right

	return border
}
