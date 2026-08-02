package decoder

import (
	"context"
	"fmt"
	"morsemanual/internal/audio"
)

type AudioToken uint8

func (t AudioToken) String() string {
	switch t {
	case CTCBlank:
		return "CTCBlank"
	case Dit:
		return "."
	case Dah:
		return "-"
	case EndCharacter:
		return " "
	case EndWord:
		return "    "
	default:
		return fmt.Sprintf("AudioToken(%d)", t)
	}
}

const (
	CTCBlank AudioToken = iota
	Dit
	Dah
	EndCharacter
	EndWord
)

const frameSamples = 160

type streaming struct {
	stft          STFT
	pending       []float32
	model         Model
	previousToken AudioToken
	morseTokens   []AudioToken
}

type Streaming interface {
	Process(
		ctx context.Context,
		chunk audio.Chunk,
		consume func(context.Context, string) error,
	) error
}

func NewStreaming() (Streaming, error) {
	model, err := LoadModel()
	if err != nil {
		return nil, err
	}
	return &streaming{
		stft:  NewSTFT(),
		model: model,
	}, nil
}

func (s *streaming) Process(
	ctx context.Context,
	chunk audio.Chunk,
	consume func(context.Context, string) error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.pending = append(s.pending, chunk.Samples...)

	for len(s.pending) >= frameSamples {
		if err := ctx.Err(); err != nil {
			return err
		}

		frame := s.pending[:frameSamples]
		s.pending = s.pending[frameSamples:]

		features, err := s.stft.Compute(frame)
		if err != nil {
			return fmt.Errorf("compute STFT: %w", err)
		}

		logits, err := s.model.Step(features)
		if err != nil {
			return fmt.Errorf("run model: %w", err)
		}

		token := AudioToken(argmax(logits))

		// Online CTC collapse.
		emit := token != s.previousToken && token != CTCBlank
		s.previousToken = token

		if !emit {
			continue
		}

		switch token {
		case Dit, Dah:
			s.morseTokens = append(s.morseTokens, token)

		case EndCharacter:
			text := s.finishCharacter()
			if text != "" {
				if err := consume(ctx, text); err != nil {
					return err
				}
			}

		case EndWord:
			text := s.finishCharacter()
			text += " "
			if err := consume(ctx, text); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *streaming) finishCharacter() string {
	if len(s.morseTokens) == 0 {
		return ""
	}

	text := decodeMorseTokens(s.morseTokens)
	s.morseTokens = s.morseTokens[:0]

	return text
}

func argmax(values []float32) int {
	bestIndex := 0
	bestValue := values[0]

	for index := 1; index < len(values); index++ {
		if values[index] > bestValue {
			bestValue = values[index]
			bestIndex = index
		}
	}

	return bestIndex
}
