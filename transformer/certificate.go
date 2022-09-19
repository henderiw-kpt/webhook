package transformer

import (
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
)

func buildCertificate(t *Webhook, obj interface{}) (*fn.KubeObject, error) {
	ns := &certv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       certv1.CertificateKind,
			APIVersion: certv1.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.getCertificateName(),
			Namespace: t.Webhook.Namespace,
		},
		Spec: certv1.CertificateSpec{
			DNSNames: []string{
				t.getDnsName(),
				t.getDnsName("cluster", "local"),
			},
			IssuerRef: certmetav1.ObjectReference{
				Kind: certv1.IssuerKind,
				Name: t.Certificate.IssuerRef,
			},
			SecretName: t.getCertificateName(),
		},
	}

	b := new(strings.Builder)
	p := printers.YAMLPrinter{}
	if err := p.PrintObj(ns, b); err != nil {
		return nil, err
	}

	return fn.ParseKubeObject([]byte(b.String()))
}
