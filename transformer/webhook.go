package transformer

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/yndd/ndd-runtime/pkg/utils"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
)

//crdObjects fn.KubeObjects
func buildMutatingWebhook(t *Webhook, obj interface{}) (*fn.KubeObject, error) {
	failurePolicy := admissionv1.Fail
	sideEffect := admissionv1.SideEffectClassNone

	webhooks := []admissionv1.MutatingWebhook{}
	objs, ok := obj.([]*fn.KubeObject)
	if !ok {
		return nil, fmt.Errorf("wrong object in buildMutatingWebhook: %v", reflect.TypeOf(objs))
	}
	for _, o := range objs {

		crdSingular := getCRDSingular(o)
		crdPlural := getCRDPlural(o)
		crdGroup := getCRDGroup(o)
		crdVersions := getCRDversions(o)

		for _, crdVersion := range crdVersions {
			webhook := admissionv1.MutatingWebhook{
				Name:                    getMutatingWebhookName(crdSingular, crdGroup),
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      t.getServiceName(),
						Namespace: t.Webhook.Namespace,
						Path:      utils.StringPtr(strings.Join([]string{"/mutate", strings.ReplaceAll(crdGroup, ".", "-"), crdVersion, crdSingular}, "-")),
					},
				},
				Rules: []admissionv1.RuleWithOperations{
					{
						Rule: admissionv1.Rule{
							APIGroups:   []string{crdGroup},
							APIVersions: []string{crdVersion},
							Resources:   []string{crdPlural},
						},
						Operations: []admissionv1.OperationType{
							admissionv1.Create,
							admissionv1.Update,
						},
					},
				},
				FailurePolicy: &failurePolicy,
				SideEffects:   &sideEffect,
			}
			webhooks = append(webhooks, webhook)
		}
	}

	ns := &admissionv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: admissionv1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        t.getMutatingWebhookName(),
			Namespace:   t.Webhook.Namespace,
			Annotations: t.getCertificateAnnotation(),
		},
		Webhooks: webhooks,
	}

	b := new(strings.Builder)
	p := printers.YAMLPrinter{}
	if err := p.PrintObj(ns, b); err != nil {
		return nil, err
	}

	return fn.ParseKubeObject([]byte(b.String()))
}

func buildValidatingWebhook(t *Webhook, obj interface{}) (*fn.KubeObject, error) {
	failurePolicy := admissionv1.Fail
	sideEffect := admissionv1.SideEffectClassNone

	webhooks := []admissionv1.ValidatingWebhook{}
	objs, ok := obj.([]*fn.KubeObject)
	if !ok {
		return nil, fmt.Errorf("wrong object in buildValidatingWebhook: %v", reflect.TypeOf(objs))
	}
	for _, o := range objs {
		crdSingular := getCRDSingular(o)
		crdPlural := getCRDPlural(o)
		crdGroup := getCRDGroup(o)
		crdVersions := getCRDversions(o)

		for _, crdVersion := range crdVersions {
			webhook := admissionv1.ValidatingWebhook{
				Name:                    getValidatingWebhookName(crdSingular, crdGroup),
				AdmissionReviewVersions: []string{"v1"},
				ClientConfig: admissionv1.WebhookClientConfig{
					Service: &admissionv1.ServiceReference{
						Name:      t.getServiceName(),
						Namespace: t.Webhook.Namespace,
						Path:      utils.StringPtr(strings.Join([]string{"/validate", strings.ReplaceAll(crdGroup, ".", "-"), crdVersion, crdSingular}, "-")),
					},
				},
				Rules: []admissionv1.RuleWithOperations{
					{
						Rule: admissionv1.Rule{
							APIGroups:   []string{crdGroup},
							APIVersions: []string{crdVersion},
							Resources:   []string{crdPlural},
						},
						Operations: []admissionv1.OperationType{
							admissionv1.Create,
							admissionv1.Update,
						},
					},
				},
				FailurePolicy: &failurePolicy,
				SideEffects:   &sideEffect,
			}
			webhooks = append(webhooks, webhook)
		}
	}

	ns := &admissionv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ValidatingWebhookConfiguration",
			APIVersion: admissionv1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        t.getValidatingWebhookName(),
			Namespace:   t.Webhook.Namespace,
			Annotations: t.getCertificateAnnotation(),
		},
		Webhooks: webhooks,
	}

	b := new(strings.Builder)
	p := printers.YAMLPrinter{}
	if err := p.PrintObj(ns, b); err != nil {
		return nil, err
	}

	return fn.ParseKubeObject([]byte(b.String()))
}

func getCRDSingular(o *fn.KubeObject) string {
	if spec := o.GetMap("spec"); spec != nil {
		if names := spec.GetMap("names"); names != nil {
			return names.GetString("singular")
		}
	}
	return ""
}

func getCRDPlural(o *fn.KubeObject) string {
	if spec := o.GetMap("spec"); spec != nil {
		if names := spec.GetMap("names"); names != nil {
			return names.GetString("plural")
		}
	}
	return ""
}

func getCRDGroup(o *fn.KubeObject) string {
	if spec := o.GetMap("spec"); spec != nil {
		return spec.GetString("group")
	}
	return ""
}

func getCRDversions(o *fn.KubeObject) []string {
	versions := []string{}
	if spec := o.GetMap("spec"); spec != nil {
		for _, vctObj := range spec.GetSlice("versions") {
			versions = append(versions, vctObj.GetString("name"))
		}
	}
	return versions
}
