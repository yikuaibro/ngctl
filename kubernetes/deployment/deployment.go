/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package deployment

import (
	"cli/util"
	"context"

	"k8s.io/klog/v2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultImage          = "vesoft/nebula-graph-studio"
	DefaultDeploymentName = "nebula-studio"
)

// NewDeploymentClient creates a new Deployment
func NewDeploymentClient(c client.Client) Deployment {
	return deploymentClient{
		getter: getter{
			Client: c,
		},
		setter: setter{
			Client: c,
		},
	}
}

// Deployment interface contains setter and getter
type Deployment interface {
	Getter
	Setter
}

type deploymentClient struct {
	getter
	setter
}

// Getter get Deployment from different parameters
type Getter interface {
	GetByNamespacedName(context.Context, types.NamespacedName) (*appsv1.Deployment, error)
}

type getter struct {
	client.Client
}

// GetByNamespacedName returns Deployment from given namespaced name
func (dg getter) GetByNamespacedName(ctx context.Context, namespacedName types.NamespacedName) (*appsv1.Deployment, error) {
	dp := &appsv1.Deployment{}
	if err := dg.Client.Get(ctx, namespacedName, dp); err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("nebula-studio deployment not found, Namespace: %s, Name: %s", namespacedName.Namespace, namespacedName.Name)
			return nil, nil
		}
		return nil, err
	} else {
		return dp, nil
	}
}

// Setter get Deployment from different parameters
type Setter interface {
	Create(context.Context, *appsv1.Deployment) error
	Update(context.Context, *appsv1.Deployment) error
}

type setter struct {
	client.Client
}

// Create creates Deployment
func (ds setter) Create(ctx context.Context, dp *appsv1.Deployment) error {
	return ds.Client.Create(ctx, dp)
}

// Update updates Deployment
func (ds setter) Update(ctx context.Context, dp *appsv1.Deployment) error {
	return ds.Client.Update(ctx, dp)
}

func DefaultDeployment() *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultDeploymentName,
			Namespace: util.DefaultNamespace,
			Labels: map[string]string{
				"app": "nebula-studio",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: util.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nebula-studio",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nebula-studio",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            DefaultDeploymentName,
							Image:           defaultImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							//Ports: []corev1.ContainerPort{
							//	{
							//		Name:          "http",
							//		ContainerPort: 7001,
							//	},
							//},
						},
					},
				},
			},
		},
	}
	return deployment
}
