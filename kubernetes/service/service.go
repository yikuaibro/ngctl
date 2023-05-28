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

package service

import (
	"cli/util"
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultServiceName = "nebula-studio"
)

// NewService creates a new Service
func NewServiceClient(c client.Client) Service {
	return serviceClient{
		getter: getter{
			Client: c,
		},
		setter: setter{
			Client: c,
		},
	}
}

// Service interface contains setter and getter
type Service interface {
	Getter
	Setter
}

// Getter get Service from different parameters
type Getter interface {
	GetByNamespacedName(context.Context, types.NamespacedName) (*corev1.Service, error)
}

// Setter set Service from different parameters
type Setter interface {
	Create(context.Context, *corev1.Service) error
	Update(context.Context, *corev1.Service) error
}

type serviceClient struct {
	getter
	setter
}

type getter struct {
	client.Client
}

// GetByNamespacedName returns a service by its namespaced name
func (sg getter) GetByNamespacedName(ctx context.Context, namespacedName types.NamespacedName) (*corev1.Service, error) {
	svc := &corev1.Service{}
	if err := sg.Get(ctx, namespacedName, svc); err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("nebula-studio service not found, Namespace: %s, Name: %s", namespacedName.Namespace, namespacedName.Name)
			return nil, nil
		}
		return nil, err
	} else {
		return svc, nil
	}
}

type setter struct {
	client.Client
}

// Create creates a service
func (sg setter) Create(ctx context.Context, svc *corev1.Service) error {
	return sg.Client.Create(ctx, svc)
}

// Update updates a service
func (sg setter) Update(ctx context.Context, svc *corev1.Service) error {
	// TODO: Compare Status with or without modification
	return sg.Client.Update(ctx, svc)
}

func DefaultService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultServiceName,
			Namespace: util.DefaultNamespace,
			Labels: map[string]string{
				"app": "nebula-studio",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "nebula-studio",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: util.DefaultPort,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}
