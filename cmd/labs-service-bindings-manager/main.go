/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/certificates"
	"knative.dev/pkg/webhook/configmaps"
	"knative.dev/pkg/webhook/psbinding"
	"knative.dev/pkg/webhook/resourcesemantics"
	"knative.dev/pkg/webhook/resourcesemantics/defaulting"
	"knative.dev/pkg/webhook/resourcesemantics/validation"

	labsv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labs/v1alpha1"
	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
	servicebindinginternalv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebindinginternal/v1alpha2"
	"github.com/vmware-labs/service-bindings/pkg/reconciler/provisionedservice"
	"github.com/vmware-labs/service-bindings/pkg/reconciler/servicebinding"
	"github.com/vmware-labs/service-bindings/pkg/reconciler/servicebindingprojection"
)

var (
	//BindingExcludeLabel can be applied to exclude resource from webhook
	BindingExcludeLabel = "bindings.labs.vmware.com/exclude"
	//BindingIncludeLabel can be applied to include resource in webhook
	BindingIncludeLabel = "bindings.labs.vmware.com/include"

	ExclusionSelector = metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      BindingExcludeLabel,
			Operator: metav1.LabelSelectorOpNotIn,
			Values:   []string{"true"},
		}},
	}
	InclusionSelector = metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      BindingIncludeLabel,
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{"true"},
		}},
	}
)
var ourTypes = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
	labsv1alpha1.SchemeGroupVersion.WithKind("ProvisionedService"):                 &labsv1alpha1.ProvisionedService{},
	servicebindingv1alpha2.SchemeGroupVersion.WithKind("ServiceBinding"):           &servicebindingv1alpha2.ServiceBinding{},
	servicebindingv1alpha2.SchemeGroupVersion.WithKind("ServiceBindingProjection"): &servicebindinginternalv1alpha2.ServiceBindingProjection{},
}

func NewDefaultingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return defaulting.NewAdmissionController(ctx,
		// Name of the resource webhook.
		"defaulting.webhook.bindings.labs.vmware.com",

		// The path on which to serve the webhook.
		"/defaulting",

		// The resources to validate and default.
		ourTypes,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return ctx
		},

		// Whether to disallow unknown fields.
		true,
	)
}

func NewValidationAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return validation.NewAdmissionController(ctx,
		// Name of the resource webhook.
		"validation.webhook.bindings.labs.vmware.com",

		// The path on which to serve the webhook.
		"/validation",

		// The resources to validate and default.
		ourTypes,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			return ctx
		},

		// Whether to disallow unknown fields.
		true,
	)
}

func NewConfigValidationController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return configmaps.NewAdmissionController(ctx,

		// Name of the configmap webhook.
		"config.webhook.bindings.labs.vmware.com",

		// The path on which to serve the webhook.
		"/config-validation",

		// The configmaps to validate.
		configmap.Constructors{
			logging.ConfigMapName(): logging.NewConfigFromConfigMap,
			metrics.ConfigMapName(): metrics.NewObservabilityConfigFromConfigMap,
		},
	)
}

func NewBindingWebhook(resource string, gla psbinding.GetListAll, wcf WithContextFactory) injection.ControllerConstructor {
	selector := psbinding.WithSelector(ExclusionSelector)
	if os.Getenv("BINDING_SELECTION_MODE") == "inclusion" {
		selector = psbinding.WithSelector(InclusionSelector)
	}
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		var wc psbinding.BindableContext
		if wcf != nil {
			wc = wcf(ctx, func(types.NamespacedName) {})
		}
		return psbinding.NewAdmissionController(ctx,
			// Name of the resource webhook.
			fmt.Sprintf("%s.webhook.bindings.labs.vmware.com", resource),

			// The path on which to serve the webhook.
			fmt.Sprintf("/%s", resource),

			// How to get all the Bindables for configuring the mutating webhook.
			gla,

			// How to setup the context prior to invoking Do/Undo.
			wc,
			selector,
		)
	}
}

func main() {
	// Set up a signal context with our webhook options
	ctx := webhook.WithOptions(signals.NewContext(), webhook.Options{
		ServiceName: "webhook",
		Port:        8443,
		SecretName:  "webhook-certs",
	})

	sharedmain.WebhookMainWithContext(ctx, "webhook",
		// Our singleton certificate controller.
		certificates.NewController,

		// Our singleton webhook admission controllers
		NewDefaultingAdmissionController,
		NewValidationAdmissionController,
		NewConfigValidationController,

		// Our reconcilers
		provisionedservice.NewController,
		servicebinding.NewController,
		servicebindingprojection.NewController, NewBindingWebhook("servicebindingprojections", servicebindingprojection.ListAll, nil),
	)
}

type WithContextFactory func(ctx context.Context, handler func(name types.NamespacedName)) psbinding.BindableContext
