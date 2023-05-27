package client

import (
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// ClientOptions the options for creating multi-cluster gatedClient
type ClientOptions struct {
	client.Options
}

func NewClient(config *rest.Config) (client.Client, error) {
	base, err := client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}
	return base, nil
}

type Args struct {
	config *rest.Config
	Client *clientset.Clientset
	client client.Client
}

func (a *Args) SetClient(c client.Client) {
	a.client = c
}

func (a *Args) GetClient() (client.Client, error) {
	if a.client != nil {
		return a.client, nil
	}
	if a.config == nil {
		if err := a.SetConfig(nil); err != nil {
			return nil, err
		}
	}
	newClient, err := NewClient(a.config)
	if err != nil {
		return nil, err
	}
	a.client = newClient
	return a.client, nil
}

func (a *Args) GetClientV1() (*clientset.Clientset, error) {
	if a.Client != nil {
		return a.Client, nil
	}
	if a.config == nil {
		if err := a.SetConfig(nil); err != nil {
			return nil, err
		}
	}
	newClient, err := clientset.NewForConfig(a.config)
	if err != nil {
		return nil, err
	}
	a.Client = newClient
	return a.Client, nil
}

func (a *Args) SetConfig(c *rest.Config) error {
	if c != nil {
		a.config = c
		return nil
	}
	restConf, err := config.GetConfig()
	if err != nil {
		return err
	}
	restConf.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 200)
	a.config = restConf
	return nil
}

func (a *Args) GetConfig() (*rest.Config, error) {
	if a.config != nil {
		return a.config, nil
	}
	if err := a.SetConfig(nil); err != nil {
		return nil, err
	}
	return a.config, nil
}
