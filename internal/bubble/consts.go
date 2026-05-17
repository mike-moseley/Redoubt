package bubble

type GameMode int

const(
	ModeNormal GameMode = iota
	ModeChat
)

const (
      paneSideW   = 2 // left + right border chars
      paneTopH    = 1 // custom-drawn ╭...╮ row
      paneBottomH = 1 // lipgloss bottom border row
      chatInputH  = 1 // textinput row
  )
