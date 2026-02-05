package layout

type Strategy interface {
	NextSplit(index int) string
}

type tiledStrategy struct {
	totalPanes int
}

type evenHorizontalStrategy struct{}
type evenVerticalStrategy struct{}
type alternatingStrategy struct{}

func NewStrategy(layoutType string, totalPanes int) Strategy {
	switch layoutType {
	case "tiled":
		return &tiledStrategy{totalPanes: totalPanes}
	case "even-horizontal":
		return &evenHorizontalStrategy{}
	case "even-vertical":
		return &evenVerticalStrategy{}
	default:
		return &alternatingStrategy{}
	}
}

func (s *alternatingStrategy) NextSplit(index int) string {
	if index%2 == 0 {
		return "vertical"
	}
	return "horizontal"
}

func (s *evenHorizontalStrategy) NextSplit(index int) string {
	return "horizontal"
}

func (s *evenVerticalStrategy) NextSplit(index int) string {
	return "vertical"
}

func (s *tiledStrategy) NextSplit(index int) string {
	if index <= s.totalPanes/2 {
		return "vertical"
	}
	return "horizontal"
}
