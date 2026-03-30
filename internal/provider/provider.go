package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/arykalin/whisper-cli/internal/domain"
)

type Request struct {
	FilePath        string
	Model           string
	Language        string
	Prompt          string
	WantDiarization bool
	WantRaw         bool
}

type Response struct {
	Transcript domain.Transcript
	Raw        []byte
}

type Client interface {
	Name() domain.Provider
	Preflight() error
	Capabilities(model string) (domain.Capabilities, bool)
	SupportedModels() []string
	Transcribe(ctx context.Context, req Request) (Response, error)
}

type Registry struct {
	clients map[domain.Provider]Client
}

func NewRegistry(clients ...Client) Registry {
	registry := Registry{
		clients: map[domain.Provider]Client{},
	}
	for _, client := range clients {
		registry.clients[client.Name()] = client
	}
	return registry
}

func (r Registry) Provider(name domain.Provider) (Client, error) {
	client, ok := r.clients[name]
	if !ok {
		return nil, fmt.Errorf("provider %s is not registered", name)
	}
	return client, nil
}

type blockedClient struct {
	name   domain.Provider
	reason error
}

func NewBlockedClient(name domain.Provider, reason error) Client {
	return blockedClient{name: name, reason: reason}
}

func (b blockedClient) Name() domain.Provider {
	return b.name
}

func (b blockedClient) Preflight() error {
	return b.reason
}

func (b blockedClient) Capabilities(string) (domain.Capabilities, bool) {
	return domain.Capabilities{}, false
}

func (b blockedClient) SupportedModels() []string {
	return nil
}

func (b blockedClient) Transcribe(context.Context, Request) (Response, error) {
	return Response{}, b.reason
}

var ErrOpenRouterPlanned = errors.New("provider openrouter is planned but not implemented yet")
