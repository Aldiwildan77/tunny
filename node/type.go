package node

type NodeType string

const (
	DirectNodeType    NodeType = "direct"
	TailscaleNodeType NodeType = "tailscale"
)

func (n NodeType) String() string {
	return string(n)
}
