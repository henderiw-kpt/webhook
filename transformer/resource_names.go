package transformer

import "strings"

const (
	webhookPrefix           = "webhook"
	serviceSuffix           = "svc"
	certSuffix              = "serving-cert"
	certPathSuffix          = "serving-certs"
	certInjectionKey        = "cert-manager.io/inject-ca-from"
	webhookMutatingSuffix   = "mutating-configuration"
	webhookValidatingSuffix = "validating-configuration"
)

func (t *Webhook) getName() string {
	return strings.Join([]string{webhookPrefix, t.Webhook.Name}, "-")
}

func (t *Webhook) getServiceName() string {
	return strings.Join([]string{t.getName(), serviceSuffix}, "-")
}

func (t *Webhook) getCertificateName() string {
	return strings.Join([]string{t.getName(), certSuffix}, "-")
}

func (t *Webhook) getMutatingWebhookName() string {
	return strings.Join([]string{t.getName(), webhookMutatingSuffix}, "-")
}

func (t *Webhook) getValidatingWebhookName() string {
	return strings.Join([]string{t.getName(), webhookValidatingSuffix}, "-")
}

func (t *Webhook) getLabelKey() string {
	return strings.Join([]string{builtinConfigMapName, webhookValidatingSuffix}, "/")
}

func (t *Webhook) getDnsName(x ...string) string {
	s := []string{t.getServiceName(), t.Webhook.Namespace, serviceSuffix}
	if len(x) > 0 {
		s = append(s, x...)
	}
	return strings.Join(s, ".")
}

func (t *Webhook) getCertificateAnnotation() map[string]string {
	return map[string]string{
		certInjectionKey: strings.Join([]string{t.Webhook.Namespace, t.getCertificateName()}, "/"),
	}
}

func getMutatingWebhookName(crdSingular, crdGroup string) string {
	return strings.Join([]string{"m" + crdSingular, crdGroup}, ".")
}

func getValidatingWebhookName(crdSingular, crdGroup string) string {
	return strings.Join([]string{"v" + crdSingular, crdGroup}, ".")
}

