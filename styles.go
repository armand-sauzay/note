package main

import (
	"github.com/charmbracelet/lipgloss"
)

type Styles struct {
	highlight lipgloss.Color
	sidebar   lipgloss.Style
	viewport  lipgloss.Style
	title     lipgloss.Style
	statusBar lipgloss.Style
	welcome   lipgloss.Style
	doc       lipgloss.Style
	header    lipgloss.Style
}

func NewStyles(cfg *Config) Styles {
	highlight := lipgloss.Color("#9D8CFF")
	paddingH, paddingV := cfg.GetPadding()
	dims := cfg.DefaultDimensions()

	return Styles{
		highlight: highlight,
		sidebar: lipgloss.NewStyle().
			Padding(paddingV, paddingH),
		viewport: lipgloss.NewStyle(),
		title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(highlight).
			Bold(true).
			Padding(0, paddingH),
		statusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Height(dims.Heights.Status).
			Padding(0, paddingH),
		doc: lipgloss.NewStyle().
			Padding(paddingH, paddingH),
		header: lipgloss.NewStyle().
			Bold(true).
			Height(dims.Heights.Header).
			Padding(0, paddingH).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#FFFFFF")),
	}
}

func (s Styles) RenderHeader(width int, dims Dimensions) func(text string) string {
	return func(text string) string {
		styledText := lipgloss.NewStyle().
			Foreground(s.highlight).
			Render(text)

		return s.header.
			Width(width).
			Height(dims.Heights.Header).
			Align(lipgloss.Center).
			BorderBottom(true).
			Render(styledText + " ✍️")
	}
}

func (s Styles) RenderSidebar(width, height int, headerGap int) func(content string) string {
	return func(content string) string {
		return s.sidebar.
			Height(height).
			Width(width).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(s.highlight).
			MarginTop(headerGap).
			Render(content)
	}
}

func (s Styles) RenderContent(width, height int, headerGap int) func(content string) string {
	return func(content string) string {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(s.highlight).
			Width(width).
			Height(height).
			MarginTop(headerGap).
			Render(content)
	}
}

func (s Styles) RenderStatusBar(width int) func(text string) string {
	return func(text string) string {
		return s.statusBar.
			Width(width).
			Render(text)
	}
}
