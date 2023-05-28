/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	argclient "cli/client"
	"cli/kubernetes/deployment"
	"cli/kubernetes/service"
	"cli/util"
	"context"
	"fmt"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/klog/v2"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"syscall"
	"time"
)

type ngCreateOptions struct {
	client client.Client
	util.IOStreams
	Namespace string
	c         argclient.Args
}

func NewCreateCommand(c argclient.Args, ioStream util.IOStreams) *cobra.Command {
	ng := &ngCreateOptions{IOStreams: ioStream, c: c}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create svc and deployment for nebula-graph",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("create called")
			ctx := context.Background()
			newClient, err := c.GetClient()
			if err != nil {
				return err
			}
			ng.client = newClient
			if err = ng.getNamespaceArg(ctx, cmd); err != nil {
				return err
			}
			if err = ng.createService(ctx, types.NamespacedName{
				Namespace: ng.Namespace,
				Name:      service.DefaultServiceName,
			}); err != nil {
				return err
			}
			if err = ng.createDeployment(ctx, types.NamespacedName{
				Namespace: ng.Namespace,
				Name:      deployment.DefaultDeploymentName,
			}); err != nil {
				return err
			}
			if err = ng.localAccess(cmd); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolP("local", "l", false, "local access to nebula-studio")
	cmd.Flags().StringP("namespace", "n", util.DefaultNamespace, "namespace of nebula-studio")
	cmd.SetOut(ioStream.Out)
	return cmd
}

func (ng *ngCreateOptions) getNamespaceArg(ctx context.Context, cmd *cobra.Command) error {
	nn, _ := cmd.Flags().GetString("namespace")
	namespace := &corev1.Namespace{}
	typenn := types.NamespacedName{Namespace: nn}
	if err := ng.client.Get(ctx, typenn, namespace); err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("%s namespace not found, ready to create", nn)
			namespace.Name = nn
			if err = ng.client.Create(ctx, namespace); err != nil {
				return err
			}
			ng.Namespace = nn
		} else {
			return err
		}
	}
	return nil
}

func (ng *ngCreateOptions) createService(ctx context.Context, nn types.NamespacedName) error {
	svcClient := service.NewServiceClient(ng.client)
	svc, err := svcClient.GetByNamespacedName(ctx, nn)
	if err != nil {
		return err
	}
	if svc == nil {
		svc = service.DefaultService()
		klog.Infof("create service, Namespace: %s, Name: %s", svc.Namespace, svc.Name)
		if err = svcClient.Create(ctx, svc); err != nil {
			return err
		}
	}
	return nil
}

func (ng *ngCreateOptions) createDeployment(ctx context.Context, nn types.NamespacedName) error {
	dpClient := deployment.NewDeploymentClient(ng.client)
	dp, err := dpClient.GetByNamespacedName(ctx, nn)
	if err != nil {
		return err
	}
	if dp == nil {
		dp = deployment.DefaultDeployment()
		if err = dpClient.Create(ctx, dp); err != nil {
			return err
		}
	}
	return nil
}

func (ng *ngCreateOptions) localAccess(cmd *cobra.Command) error {
	local, _ := cmd.Flags().GetBool("local")

	if local {
		newclient, err := ng.c.GetClientV1()
		if err != nil {
			return err
		}
		svc, err := newclient.CoreV1().Services(util.DefaultNamespace).Get(context.Background(), service.DefaultServiceName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		selector, err := labels.ValidatedSelectorFromSet(svc.Spec.Selector)
		if err != nil {
			return err
		}

		pod := &corev1.Pod{}
		err = waitForPodRunning(newclient, pod, svc.Namespace, selector.String(), 5*time.Second, 1*time.Minute)

		klog.Info(pod.Name)

		req := newclient.CoreV1().RESTClient().Post().Namespace(pod.Namespace).
			Resource("pods").Name(pod.Name).SubResource("portforward")
		klog.Info(req.URL())

		signals := make(chan os.Signal, 1)
		StopChannel := make(chan struct{}, 1)
		ReadyChannel := make(chan struct{})

		defer signal.Stop(signals)

		signal.Notify(signals, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

		go func() {
			<-signals
			if StopChannel != nil {
				close(StopChannel)
			}
		}()

		config, err := ng.c.GetConfig()
		if err != nil {
			return err
		}
		if err := ng.ForwardPorts("POST", req.URL(), config, StopChannel, ReadyChannel); err != nil {
			klog.Fatalln(err)
		}
	}
	return nil
}
func (ng *ngCreateOptions) ForwardPorts(method string, url *url.URL, config *rest.Config, StopChannel, ReadyChannel chan struct{}) error {
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return err
	}
	//address := []string{"0.0.0.0"}
	ports := []string{fmt.Sprintf("%d:%d", util.DefaultPort, util.DefaultPort)}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, method, url)
	fw, err := portforward.New(dialer, ports, StopChannel, ReadyChannel, ng.Out, ng.ErrOut)
	if err != nil {
		return err
	}
	klog.Info(fmt.Sprintf("nebula-studio is running on http://localhost:%d", util.DefaultPort))
	return fw.ForwardPorts()
}

func waitForPodRunning(client *kubernetes.Clientset, pod *corev1.Pod, namespace, selector string,
	waitInterval, timeout time.Duration) error {

	return wait.PollUntilContextTimeout(context.Background(), waitInterval, timeout, true,
		func(ctx context.Context) (bool, error) {
			podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
			if err != nil {
				return false, err
			}
			for i, _ := range podList.Items {
				if podList.Items[i].Status.Phase == corev1.PodRunning {
					*pod = podList.Items[i]
					klog.Info(fmt.Sprintf("pod %s is running", pod.Name))
					return true, nil
				}
			}
			return false, nil
		})

}
