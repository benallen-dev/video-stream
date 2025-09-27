package channel

type playerState int

const (
	PlayerStopped playerState = iota
	PlayerPlaying
)

var playerStateName = map[playerState]string{
	PlayerStopped: "stopped",
	PlayerPlaying: "playing",
}

func (ps playerState) String() string {
	return playerStateName[ps]
}
