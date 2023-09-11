package itch

import (
	"io"
)

type Processor struct {
	handler        Handler
	unmarshalFuncs [256]func([]byte) error
	msgLength      int
	cache          []byte
}

func NewProcessor(handler Handler) (*Processor, error) {
	processor := &Processor{
		handler: handler,
		cache:   make([]byte, 0, 256*256+2),
	}
	err := processor.initialize()
	if err != nil {
		return nil, err
	}
	return processor, nil
}

func (p *Processor) Process(reader io.Reader) (err error) {
	chunk := [1024 * 1024]byte{}
	for readBytes := 0; err != io.EOF; {
		// Read chunk bytes
		readBytes, err = reader.Read(chunk[:])
		if err != nil && err != io.EOF {
			return err
		}
		// Process the chunk
		if err := p.ProcessChunk(chunk[:readBytes]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Processor) ProcessChunk(chunk []byte) (err error) {

	for offset, size := 0, len(chunk); offset < size; {

		if p.msgLength == 0 {
			remaining := size - offset

			// Collect message size into the cache
			if (len(p.cache) == 0 && remaining < 3) || len(p.cache) == 1 {
				p.cache = append(p.cache, chunk[offset])
				offset++
				continue
			}

			// Read a new message size
			var msgLength uint16
			if len(p.cache) == 0 {
				// Read the message size directly from the input buffer
				msgLength, _ = readUint16(chunk[offset : offset+2])
				offset += 2
			} else {
				// Read the message size from the cache
				msgLength, _ = readUint16(p.cache[:2])
				// Clear the cache
				p.cache = p.cache[:0]
			}
			p.msgLength = int(msgLength)
		}

		// Read a new message
		if p.msgLength > 0 {
			remaining := size - offset

			// Complete or place the message into the cache
			if len(p.cache) > 0 {
				tail := p.msgLength - len(p.cache)
				if tail > remaining {
					tail = remaining
				}
				p.cache = append(p.cache, chunk[offset:offset+tail]...)
				offset += tail
				if p.msgLength > len(p.cache) {
					continue
				}
			} else if p.msgLength > remaining {
				p.cache = append(p.cache, chunk[offset:offset+remaining]...)
				offset += remaining
				continue
			}

			// Process the current message
			if len(p.cache) == 0 {
				// Process the current message size directly from the input buffer
				err = p.unmarshalFuncs[chunk[offset]](chunk[offset : offset+p.msgLength])
				if err != nil {
					return err
				}
				offset += p.msgLength
			} else {
				// Process the current message size directly from the cache
				err = p.unmarshalFuncs[p.cache[0]](p.cache[:p.msgLength])
				if err != nil {
					return err
				}
				// Clear the cache
				p.cache = p.cache[:0]
			}

			// Process the next message
			p.msgLength = 0
		}
	}

	return nil
}

func (p *Processor) initialize() error {
	p.unmarshalFuncs['S'] = func(data []byte) error {
		msg, err := unmarshalSystemEventMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnSystemEventMessage(msg)
		return nil
	}
	p.unmarshalFuncs['R'] = func(data []byte) error {
		msg, err := unmarshalStockDirectoryMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnStockDirectoryMessage(msg)
		return nil
	}
	p.unmarshalFuncs['H'] = func(data []byte) error {
		msg, err := unmarshalStockTradingActionMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnStockTradingActionMessage(msg)
		return nil
	}
	p.unmarshalFuncs['Y'] = func(data []byte) error {
		msg, err := unmarshalRegSHOMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnRegSHOMessage(msg)
		return nil
	}
	p.unmarshalFuncs['L'] = func(data []byte) error {
		msg, err := unmarshalMarketParticipantPositionMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnMarketParticipantPositionMessage(msg)
		return nil
	}
	p.unmarshalFuncs['V'] = func(data []byte) error {
		msg, err := unmarshalMWCBDeclineMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnMWCBDeclineMessage(msg)
		return nil
	}
	p.unmarshalFuncs['W'] = func(data []byte) error {
		msg, err := unmarshalMWCBStatusMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnMWCBStatusMessage(msg)
		return nil
	}
	p.unmarshalFuncs['K'] = func(data []byte) error {
		msg, err := unmarshalIPOQuotingMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnIPOQuotingMessage(msg)
		return nil
	}
	p.unmarshalFuncs['A'] = func(data []byte) error {
		msg, err := unmarshalAddOrderMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnAddOrderMessage(msg)
		return nil
	}
	p.unmarshalFuncs['F'] = func(data []byte) error {
		msg, err := unmarshalAddOrderMPIDMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnAddOrderMPIDMessage(msg)
		return nil
	}
	p.unmarshalFuncs['E'] = func(data []byte) error {
		msg, err := unmarshalOrderExecutedMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnOrderExecutedMessage(msg)
		return nil
	}
	p.unmarshalFuncs['C'] = func(data []byte) error {
		msg, err := unmarshalOrderExecutedWithPriceMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnOrderExecutedWithPriceMessage(msg)
		return nil
	}
	p.unmarshalFuncs['X'] = func(data []byte) error {
		msg, err := unmarshalOrderCancelMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnOrderCancelMessage(msg)
		return nil
	}
	p.unmarshalFuncs['D'] = func(data []byte) error {
		msg, err := unmarshalOrderDeleteMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnOrderDeleteMessage(msg)
		return nil
	}
	p.unmarshalFuncs['U'] = func(data []byte) error {
		msg, err := unmarshalOrderReplaceMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnOrderReplaceMessage(msg)
		return nil
	}
	p.unmarshalFuncs['P'] = func(data []byte) error {
		msg, err := unmarshalTradeMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnTradeMessage(msg)
		return nil
	}
	p.unmarshalFuncs['Q'] = func(data []byte) error {
		msg, err := unmarshalCrossTradeMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnCrossTradeMessage(msg)
		return nil
	}
	p.unmarshalFuncs['B'] = func(data []byte) error {
		msg, err := unmarshalBrokenTradeMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnBrokenTradeMessage(msg)
		return nil
	}
	p.unmarshalFuncs['I'] = func(data []byte) error {
		msg, err := unmarshalNOIIMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnNOIIMessage(msg)
		return nil
	}
	p.unmarshalFuncs['N'] = func(data []byte) error {
		msg, err := unmarshalRPIIMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnRPIIMessage(msg)
		return nil
	}
	p.unmarshalFuncs['J'] = func(data []byte) error {
		msg, err := unmarshalLULDAuctionCollarMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnLULDAuctionCollarMessage(msg)
		return nil
	}
	// All other message types are unknown:
	unknownUnmarshalFunc := func(data []byte) error {
		msg, err := unmarshalUnknownMessage(data)
		if err != nil {
			return err
		}
		p.handler.OnUnknownMessage(msg)
		return nil
	}
	for i := 0; i < 256; i++ {
		if p.unmarshalFuncs[i] == nil {
			p.unmarshalFuncs[i] = unknownUnmarshalFunc
		}
	}
	return nil
}
