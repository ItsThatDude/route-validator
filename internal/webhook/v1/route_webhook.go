/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"fmt"

	"antware.xyz/route-validator/internal/config"
	"antware.xyz/route-validator/internal/validation"

	routeopenshiftiov1 "github.com/openshift/api/route/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var routelog = logf.Log.WithName("route-resource")

// SetupRouteWebhookWithManager registers the webhook for Route in the manager.
func SetupRouteWebhookWithManager(mgr ctrl.Manager, cfg *config.ConfigManager) error {
	validator := RouteCustomValidator{
		client: mgr.GetClient(),
		validator: validation.Validator{
			Config: cfg.Get(),
		},
	}
	return ctrl.NewWebhookManagedBy(mgr).For(&routeopenshiftiov1.Route{}).
		WithValidator(&validator).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-route-openshift-io-v1-route,mutating=false,failurePolicy=fail,sideEffects=None,groups=route.openshift.io,resources=routes,verbs=create;update,versions=v1,name=route-validator.antware.xyz,admissionReviewVersions=v1

// RouteCustomValidator struct is responsible for validating the Route resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type RouteCustomValidator struct {
	client    client.Client
	validator validation.Validator
}

var _ webhook.CustomValidator = &RouteCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Route.
func (v *RouteCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	route, ok := obj.(*routeopenshiftiov1.Route)
	if !ok {
		return nil, fmt.Errorf("expected a Route object but got %T", obj)
	}

	return v.validateCreateOrUpdate(ctx, route)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Route.
func (v *RouteCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	route, ok := newObj.(*routeopenshiftiov1.Route)
	if !ok {
		return nil, fmt.Errorf("expected a Route object for the newObj but got %T", newObj)
	}

	return v.validateCreateOrUpdate(ctx, route)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Route.
func (v *RouteCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	_, ok := obj.(*routeopenshiftiov1.Route)
	if !ok {
		return nil, fmt.Errorf("expected a Route object but got %T", obj)
	}

	return nil, nil
}

func (v *RouteCustomValidator) validateCreateOrUpdate(ctx context.Context, route *routeopenshiftiov1.Route) (admission.Warnings, error) {
	namespace := &v1.Namespace{}
	err := v.client.Get(ctx, client.ObjectKey{Name: route.Namespace}, namespace)

	if err != nil {
		return nil, fmt.Errorf("could not get namespace: %v", err)
	}

	selector, err := metav1.LabelSelectorAsSelector(v.validator.Config.NamespaceSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse namespace selector: %v", err)
	}

	// If the selector doesnt match this namespace, allow the route
	if !selector.Matches(labels.Set(namespace.Labels)) {
		return nil, nil
	}

	// If there are no MatchDomains configured, allow the route
	if len(v.validator.Config.MatchDomains) == 0 {
		return nil, nil
	}

	hostnames := []string{route.Spec.Host}
	matches := validation.MatchesAnyDomain(hostnames, v.validator.Config.MatchDomains)
	// If the route matches any of the MatchDomains, validate the hostname of the route
	if matches {
		_, err := v.validator.ValidateHostnames(namespace, hostnames)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}
