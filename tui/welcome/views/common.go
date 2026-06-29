package views

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/toanle/synthspec/tui/shared"
)

const selectionFormat = " %s %s"

// RenderLogo renders the SynthSpec ASCII art logo
func RenderLogo() string {
	logo := `
   _____             __  __   _____                     
  / ____|           |  \/  | / ____|                    
 | (___   _   _  _  | \  / || (___   _ __    ___   ___  
  \___ \ | | | || |_| |\/| | \___ \ | '_ \  / _ \ / __| 
  ____) || |_| ||  _| |  | | ____) || |_) ||  __/| (__  
 |_____/  \__, ||_| |_|  |_||_____/ | .__/  \___| \___| 
           __/ |                    | |                 
          |___/                     |_|                 
`
	logoStyle := lipgloss.NewStyle().Foreground(shared.ColorAccent).Bold(true)
	return logoStyle.Render(logo)
}
