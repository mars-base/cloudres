package tui

import "strings"

var logoLines = []string{
	"  ____ _                 _ ____  _____ ____  ",
	" / ___| | ___  _   _  __| |  _ \\| ____/ ___|",
	"| |   | |/ _ \\| | | |/ _` | |_) |  _| \\___ \\",
	"| |___| | (_) | |_| | (_| |  _ <| |___ ___) |",
	" \\____|_|\\___/ \\__,_|\\__,_|_| \\_\\_____|____/",
	"",
	"  Cloud Resource Dashboard",
}

func (m *appModel) viewLogo() string {
	var sb strings.Builder
	for _, line := range logoLines {
		sb.WriteString(colHeaderStyle.Render(line))
		sb.WriteByte('\n')
	}
	return sb.String()
}
