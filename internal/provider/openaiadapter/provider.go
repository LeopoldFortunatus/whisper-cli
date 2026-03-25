package openaiadapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/arykalin/whisper-cli/internal/domain"
	"github.com/arykalin/whisper-cli/internal/platform/fsx"
	"github.com/arykalin/whisper-cli/internal/provider"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared/constant"
	"github.com/rs/zerolog"
)

const diarizeModel = "gpt-4o-transcribe-diarize"

type requester interface {
	Transcribe(ctx context.Context, params openai.AudioTranscriptionNewParams) (*openai.Transcription, error)
}

type serviceRequester struct {
	service openai.AudioTranscriptionService
}

func (s serviceRequester) Transcribe(ctx context.Context, params openai.AudioTranscriptionNewParams) (*openai.Transcription, error) {
	return s.service.New(ctx, params)
}

type Provider struct {
	apiKey    string
	fs        fsx.FS
	requester requester
	logger    zerolog.Logger
}

func New(apiKey string, fs fsx.FS, logger zerolog.Logger) *Provider {
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}

	client := openai.NewClient(opts...)
	return &Provider{
		apiKey:    apiKey,
		fs:        fs,
		requester: serviceRequester{service: client.Audio.Transcriptions},
		logger:    logger.With().Str("provider", string(domain.ProviderOpenAI)).Logger(),
	}
}

func newWithRequester(apiKey string, fs fsx.FS, logger zerolog.Logger, requester requester) *Provider {
	return &Provider{
		apiKey:    apiKey,
		fs:        fs,
		requester: requester,
		logger:    logger.With().Str("provider", string(domain.ProviderOpenAI)).Logger(),
	}
}

func (p *Provider) Name() domain.Provider {
	return domain.ProviderOpenAI
}

func (p *Provider) Preflight() error {
	if p.apiKey == "" {
		return errors.New("OPENAI_API_KEY is not set in process environment; run `export OPENAI_API_KEY=...` or prefix the command with `OPENAI_API_KEY=...`")
	}
	return nil
}

func (p *Provider) Capabilities(model string) (domain.Capabilities, bool) {
	caps, ok := capabilities[model]
	return caps, ok
}

func (p *Provider) Transcribe(ctx context.Context, req provider.Request) (provider.Response, error) {
	caps, ok := p.Capabilities(req.Model)
	if !ok {
		return provider.Response{}, fmt.Errorf("model %s is not supported by provider %s", req.Model, p.Name())
	}

	var (
		raw  []byte
		text string
	)

	err := provider.Retry(ctx, p.logger, string(p.Name()), func() (err error) {
		file, err := p.fs.Open(req.FilePath)
		if err != nil {
			return fmt.Errorf("open audio file: %w", err)
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = fmt.Errorf("close audio file: %w", closeErr)
			}
		}()

		params := openai.AudioTranscriptionNewParams{
			File:     file,
			Model:    req.Model,
			Language: param.NewOpt(req.Language),
		}
		if req.Prompt != "" && caps.SupportsPrompt {
			params.Prompt = param.NewOpt(req.Prompt)
		}

		if req.WantDiarization {
			params.ResponseFormat = openai.AudioResponseFormat("diarized_json")
			params.ChunkingStrategy = openai.AudioTranscriptionNewParamsChunkingStrategyUnion{
				OfAuto: constant.ValueOf[constant.Auto](),
			}
		} else if caps.SupportsSegmentTimestamps {
			params.ResponseFormat = openai.AudioResponseFormatVerboseJSON
			params.TimestampGranularities = []string{"segment"}
		} else {
			params.ResponseFormat = openai.AudioResponseFormatJSON
		}

		resp, err := p.requester.Transcribe(ctx, params)
		if err != nil {
			return err
		}

		text = resp.Text
		raw = []byte(resp.RawJSON())
		return nil
	})
	if err != nil {
		return provider.Response{}, err
	}

	transcript := provider.ParseOpenAICompatibleTranscript(p.Name(), req.Model, raw, text)
	return provider.Response{
		Transcript: transcript,
		Raw:        raw,
	}, nil
}

var capabilities = map[string]domain.Capabilities{
	openai.AudioModelWhisper1: {
		SupportsPrompt:            true,
		SupportsSegmentTimestamps: true,
		SupportsSRT:               true,
		SupportsVTT:               true,
	},
	openai.AudioModelGPT4oTranscribe: {
		SupportsPrompt: true,
	},
	openai.AudioModelGPT4oMiniTranscribe: {
		SupportsPrompt: true,
	},
	diarizeModel: {
		SupportsDiarization: true,
	},
}
