package matching

const stopPriceModesCount = 3

type StopPriceModeConfig struct {
	Market bool
	Mark   bool
	Index  bool
}

func (c StopPriceModeConfig) Modes() []StopPriceMode {
	modes := make([]StopPriceMode, 0, stopPriceModesCount)

	if c.Market {
		modes = append(modes, StopPriceModeMarket)
	}

	if c.Mark {
		modes = append(modes, StopPriceModeMark)
	}

	if c.Index {
		modes = append(modes, StopPriceModeIndex)
	}

	return modes
}

type StopPriceMode uint8

const (
	StopPriceModeMarket StopPriceMode = iota + 1
	StopPriceModeMark
	StopPriceModeIndex
)

func (m StopPriceMode) String() string {
	switch m {
	case StopPriceModeMarket:
		return "market"
	case StopPriceModeMark:
		return "mark"
	case StopPriceModeIndex:
		return "index"
	default:
		return "unknown"
	}
}
