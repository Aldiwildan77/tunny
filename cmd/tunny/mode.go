package main

type Mode string

const (
	ModeProvider Mode = "provider"
	ModeProxy    Mode = "proxy"
	ModeTunnel   Mode = "tunnel"
)

func (m Mode) String() string {
	return string(m)
}
