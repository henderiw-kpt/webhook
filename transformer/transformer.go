package transformer

import (
	"fmt"
	"sort"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// Constants for FunctionConfig `SetNamespace`
	fnConfigGroup   = "fn.kpt.dev"
	fnConfigVersion = "v1alpha1"
	fnConfigKind    = "Webhook"
	// The ConfigMap name generated from variant constructor
	builtinConfigMapName = "kptfile.kpt.dev"
)

type WebHookOperation string

const (
	WebHookOperationAdd    WebHookOperation = "add"
	WebHookOperationDelete WebHookOperation = "delete"
)

// Webhook contains the information to perform the operation to create or delete a webhook of a package
type Webhook struct {
	Operation WebHookOperation `json:"operation,omitempty" yaml:"operation,omitempty"`
	// webhook meta
	Webhook WebhookMeta `json:"webhook,omitempty" yaml:"webhook,omitempty"`
	// service
	Service Service `json:"service,omitempty" yaml:"service,omitempty"`
	// sertifcate
	Certificate Certificate `json:"certificate,omitempty" yaml:"certificate,omitempty"`
	// container
	Container Container `json:"container,omitempty" yaml:"container,omitempty"`
	// sWebhookResults is used internally to track which resources were updated
	WebhookResults webhookResults
}

type WebhookMeta struct {
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

type Service struct {
	Port       int32 `json:"port,omitempty" yaml:"port,omitempty"`
	TargetPort int32 `json:"targetPort,omitempty" yaml:"targetPort,omitempty"`
}

type Certificate struct {
	IssuerRef string `json:"issuerRef,omitempty" yaml:"issuerRef,omitempty"`
}

type Container struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

// webhookResultKey is used as a unique identifier for webhook results
type webhookResultKey struct {
	ResourceRef fn.ResourceRef
	// FilePath is the file path of the resource
	FilePath string
	// FileIndex is the file index of the resource
	FileIndex int
	// FieldPath is field path of the serviceaccount field
	FieldPath string
}

// webhookResult tracks the operation
type webhookResult struct {
	Operation string
}

// setImageResults tracks the number of images updated matching the key
type webhookResults map[webhookResultKey][]webhookResult

func Run(rl *fn.ResourceList) (bool, error) {
	tc := Webhook{}
	if err := tc.config(rl.FunctionConfig); err != nil {
		rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, rl.FunctionConfig))
	}
	if err := tc.validate(); err != nil {
		rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, rl.FunctionConfig))
	}
	tc.Transform(rl)
	return true, nil
}

func (t *Webhook) config(fc *fn.KubeObject) error {
	switch {
	case fc.IsEmpty():
		return fmt.Errorf("FunctionConfig is missing. Expect `ConfigMap.v1` or `%s.%s.%s`",
			fnConfigKind, fnConfigVersion, fnConfigGroup)

	case fc.IsGVK("", "v1", "ConfigMap"):
		var cm corev1.ConfigMap
		fc.AsOrDie(&cm)

		if cm.Data["webhook"] != "" {
			if err := yaml.Unmarshal([]byte(cm.Data["webhook"]), t); err != nil {
				return fmt.Errorf("cannot marshal fucntionconfig configmap: %s", err.Error())
			}
		}

	case fc.IsGVK(fnConfigGroup, fnConfigVersion, fnConfigKind):
		fc.AsOrDie(&t)
	default:
		return fmt.Errorf("unknown functionConfig Kind=%v ApiVersion=%v, expect `ConfigMap.v1` or `%s.%s.%s`",
			fc.GetKind(), fc.GetAPIVersion(), fnConfigKind, fnConfigVersion, fnConfigGroup)
	}
	return nil
}

func (t *Webhook) validate() error {
	if err := t.validateMeta(); err != nil {
		return err
	}
	if err := t.validateContainer(); err != nil {
		return err
	}
	switch t.Operation {
	case WebHookOperationDelete:
		return nil
	case WebHookOperationAdd:
		if err := t.validateCertificateName(); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("operation should be %s or %s, got %s", WebHookOperationAdd, WebHookOperationDelete, t.Operation)
	}
}

func (t *Webhook) validateMeta() error {
	if t.Webhook.Name == "" {
		return fmt.Errorf("webhook name is required")
	}
	return nil
}

func (t *Webhook) validateContainer() error {
	if t.Webhook.Name == "" {
		return fmt.Errorf("webhook container is required")
	}
	return nil
}

func (t *Webhook) validateCertificateName() error {
	if t.Certificate.IssuerRef == "" {
		return fmt.Errorf("webhook certificate issueRefName is required")
	}
	return nil
}

type objInfo struct {
	found      bool
	idx        int
	buildObjFn func(*Webhook, interface{}) (*fn.KubeObject, error)
	objName    string
}

func (t *Webhook) Transform(rl *fn.ResourceList) {
	var results fn.Results
	if t.WebhookResults == nil {
		t.WebhookResults = webhookResults{}
	}

	matchResources := map[string]*objInfo{
		"Service": {
			buildObjFn: buildService,
			objName:    t.getServiceName(),
		},
		"Certificate": {
			buildObjFn: buildCertificate,
			objName:    t.getCertificateName(),
		},
		"MutatingWebhookConfiguration": {
			buildObjFn: buildMutatingWebhook,
			objName:    t.getMutatingWebhookName(),
		},
		"ValidatingWebhookConfiguration": {
			buildObjFn: buildValidatingWebhook,
			objName:    t.getValidatingWebhookName(),
		},
	}
	containerInfo := &objInfo{}
	crdObjects := make([]*fn.KubeObject, 0)
	for i, o := range rl.Items {
		switch o.GetKind() {
		case "CustomResourceDefinition":
			crdObjects = append(crdObjects, o)
		case "ReplicationController", "Deployment", "ReplicaSet", "StatefulSet", "DaemonSet":
			if t.validatePodContainer(o) {
				// TBD what todo if already found, we would expect 1 container
				containerInfo.found = true
				containerInfo.idx = i
			}
		default:
			if _, ok := matchResources[o.GetKind()]; ok {
				if o.GetName() == matchResources[o.GetKind()].objName {
					matchResources[o.GetKind()].found = true
					matchResources[o.GetKind()].idx = i
				}
			}
		}
	}

	// if container is not found the processing should not be executed
	if !containerInfo.found {
		results = append(results, fn.ErrorResult(fmt.Errorf("container not found")))
		rl.Results = append(rl.Results, results...)
		return
	}

	switch t.Operation {
	case WebHookOperationAdd:
		// TODO add volumes and volumemounts in container
		for _, objInfo := range matchResources {
			o, err := objInfo.buildObjFn(t, crdObjects)
			if err != nil {
				results = append(results, fn.ErrorResult(err))
			} else {
				if objInfo.found {
					rl.Items[objInfo.idx] = o
				} else {
					rl.Items = append(rl.Items, o)
				}
			}
		}
		// Add the certificate volume to the pod and volumemount to the container
		var err error
		rl.Items[containerInfo.idx], err = t.setVolumes(rl.Items[containerInfo.idx])
		if err != nil {
			results = append(results, fn.ErrorConfigObjectResult(err, rl.Items[containerInfo.idx]))
		}

	case WebHookOperationDelete:
		// sort the list so that we first delete the last entries
		keys := []int{}
		for _, objInfo := range matchResources {
			if objInfo.found {
				keys = append(keys, objInfo.idx)
			}
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] > keys[j]
		})
		for _, idx := range keys {
			rl.Items = append(rl.Items[:idx], rl.Items[idx+1:]...)
		}
	}

	rl.Results = append(rl.Results, results...)
}

func (t *Webhook) validatePodContainer(o *fn.KubeObject) bool {
	if spec := o.GetMap("spec"); spec != nil {
		if template := spec.GetMap("template"); template != nil {
			if podSpec := template.GetMap("spec"); podSpec != nil {
				for _, container := range podSpec.GetSlice("containers") {
					if container.GetString("name") == t.Container.Name {
						return true
					}
				}
			}
		}
	}
	return false
}
