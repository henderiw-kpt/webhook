package transformer

import (
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/cli-runtime/pkg/printers"
)

func buildService(t *Webhook, obj interface{}) (*fn.KubeObject, error) {
	ns := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: corev1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.getServiceName(),
			Namespace: t.Webhook.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				t.getLabelKey(): t.getServiceName(),
			},
			Ports: []corev1.ServicePort{
				{
					Name:       webhookPrefix,
					Port:       t.Service.Port,
					TargetPort: intstr.FromInt(int(t.Service.TargetPort)),
					Protocol:   corev1.Protocol("TCP"),
				},
			},
		},
	}

	b := new(strings.Builder)
	p := printers.YAMLPrinter{}
	if err := p.PrintObj(ns, b); err != nil {
		return nil, err
	}

	return fn.ParseKubeObject([]byte(b.String()))
}
