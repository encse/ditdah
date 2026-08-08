package decoder

import (
	"context"
	"fmt"
	"morsemanual/internal/audio"
	"morsemanual/internal/decoder/model"
	"morsemanual/internal/decoder/model/core"
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
const decoderCount = 3

var decoderOffsets = [decoderCount]int{
	0,
	frameSamples / 3,
	frameSamples / 2,
}

type decoderLane struct {
	stft    STFT
	model   core.Model
	skip    int
	pending []float32
	tokens  []AudioToken
}

type streaming struct {
	lanes         [decoderCount]decoderLane
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
	stream := &streaming{}

	for index := range stream.lanes {
		loadedModel, err := model.LoadModel()
		if err != nil {
			return nil, fmt.Errorf("load decoder %d: %w", index, err)
		}

		stream.lanes[index] = decoderLane{
			stft:  NewSTFT(),
			model: loadedModel,
			skip:  decoderOffsets[index],
		}
	}

	return stream, nil
}

func (s *streaming) Process(
	ctx context.Context,
	chunk audio.Chunk,
	consume func(context.Context, string) error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	for index := range s.lanes {
		if err := s.lanes[index].process(ctx, chunk.Samples); err != nil {
			return fmt.Errorf("decoder %d: %w", index, err)
		}
	}

	for s.hasVote() {
		if err := ctx.Err(); err != nil {
			return err
		}

		token := s.nextVote()

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

func (l *decoderLane) process(ctx context.Context, samples []float32) error {
	if l.skip > 0 {
		skipped := min(l.skip, len(samples))
		l.skip -= skipped
		samples = samples[skipped:]
	}

	l.pending = append(l.pending, samples...)

	for len(l.pending) >= frameSamples {
		if err := ctx.Err(); err != nil {
			return err
		}

		frame := l.pending[:frameSamples]
		l.pending = l.pending[frameSamples:]

		features, err := l.stft.Compute(frame)
		if err != nil {
			return fmt.Errorf("compute STFT: %w", err)
		}

		logits, err := l.model.Step(features)
		if err != nil {
			return fmt.Errorf("run model: %w", err)
		}

		l.tokens = append(l.tokens, AudioToken(core.Argmax(logits)))
	}

	return nil
}

func (s *streaming) hasVote() bool {
	for index := range s.lanes {
		if len(s.lanes[index].tokens) == 0 {
			return false
		}
	}
	return true
}

func (s *streaming) nextVote() AudioToken {
	var tokens [decoderCount]AudioToken
	for index := range s.lanes {
		tokens[index] = s.lanes[index].tokens[0]
		s.lanes[index].tokens = s.lanes[index].tokens[1:]
	}
	return majorityToken(tokens)
}

func majorityToken(tokens [decoderCount]AudioToken) AudioToken {
	for _, candidate := range tokens {
		votes := 0
		for _, token := range tokens {
			if token == candidate {
				votes++
			}
		}
		if votes > decoderCount/2 {
			return candidate
		}
	}

	for _, token := range tokens {
		if token == CTCBlank {
			return CTCBlank
		}
	}

	return tokens[decoderCount/2]
}

func (s *streaming) finishCharacter() string {
	if len(s.morseTokens) == 0 {
		return ""
	}

	text := decodeMorseTokens(s.morseTokens)
	s.morseTokens = s.morseTokens[:0]

	return text
}
