package transformer

import (
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/yndd/ndd-runtime/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	sigsyaml "sigs.k8s.io/yaml"
)

func (t *Webhook) setVolumes(o *fn.KubeObject) (*fn.KubeObject, error) {
	node, err := yaml.Parse(o.String())
	if err != nil {
		return nil, err
	}

	volumes, err := t.buildVolumes()
	if err != nil {
		return nil, err
	}

	volumeMounts, err := t.buildVolumeMounts()
	if err != nil {
		return nil, err
	}

	_, err = node.Pipe(
		yaml.LookupCreate(yaml.SequenceNode, "spec", "template", "spec", "volumes"),
		yaml.Append(volumes.YNode().Content...),
	)
	if err != nil {
		return nil, err
	}

	_, err = node.Pipe(
		yaml.LookupCreate(yaml.SequenceNode, "spec", "template", "spec", "containers", "[name=manager]", "volumeMounts"),
		yaml.Append(volumeMounts.YNode().Content...),
	)
	if err != nil {
		return nil, err
	}

	str, err := node.String()
	if err != nil {
		return nil, err
	}
	return fn.ParseKubeObject([]byte(str))
}

func (t *Webhook) buildVolumes() (*yaml.RNode, error) {
	v := []*corev1.Volume{
		{
			Name: t.getName(),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  t.getCertificateName(),
					DefaultMode: utils.Int32Ptr(420),
				},
			},
		},
	}

	b, err := sigsyaml.Marshal(v)
	if err != nil {
		return nil, err
	}

	return yaml.Parse(string(b))
}

func (t *Webhook) buildVolumeMounts() (*yaml.RNode, error) {
	v := []corev1.VolumeMount{
		{
			Name:      t.getName(),
			MountPath: filepath.Join("tmp", strings.Join([]string{"k8s", webhookPrefix, "server"}, "-"), certPathSuffix),
			ReadOnly:  true,
		},
	}
	b, err := sigsyaml.Marshal(v)
	if err != nil {
		return nil, err
	}

	return yaml.Parse(string(b))
}
