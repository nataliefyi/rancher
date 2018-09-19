package bitbucket

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/rancher/norman/store/subtype"
	"github.com/rancher/norman/types"
	"github.com/rancher/norman/types/convert"
	"github.com/rancher/rancher/pkg/pipeline/remote/model"
	"github.com/rancher/types/apis/project.cattle.io/v3"
	"github.com/rancher/types/apis/project.cattle.io/v3/schema"
	"github.com/rancher/types/client/project/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type BitbucketProvider struct {
	SourceCodeProviderConfigs  v3.SourceCodeProviderConfigInterface
	SourceCodeCredentials      v3.SourceCodeCredentialInterface
	SourceCodeCredentialLister v3.SourceCodeCredentialLister
	SourceCodeRepositories     v3.SourceCodeRepositoryInterface
	Pipelines                  v3.PipelineInterface
	PipelineExecutions         v3.PipelineExecutionInterface

	PipelineIndexer             cache.Indexer
	PipelineExecutionIndexer    cache.Indexer
	SourceCodeCredentialIndexer cache.Indexer
	SourceCodeRepositoryIndexer cache.Indexer
}

func (b *BitbucketProvider) CustomizeSchemas(schemas *types.Schemas) {
	scpConfigBaseSchema := schemas.Schema(&schema.Version, client.SourceCodeProviderConfigType)
	configSchema := schemas.Schema(&schema.Version, client.BitbucketCloudPipelineConfigType)
	configSchema.ActionHandler = b.ActionHandler
	configSchema.Formatter = b.Formatter
	configSchema.Store = subtype.NewSubTypeStore(client.BitbucketCloudPipelineConfigType, scpConfigBaseSchema.Store)

	providerBaseSchema := schemas.Schema(&schema.Version, client.SourceCodeProviderType)
	providerSchema := schemas.Schema(&schema.Version, client.BitbucketCloudProviderType)
	providerSchema.Formatter = b.providerFormatter
	providerSchema.ActionHandler = b.providerActionHandler
	providerSchema.Store = subtype.NewSubTypeStore(client.BitbucketCloudProviderType, providerBaseSchema.Store)
}

func (b *BitbucketProvider) GetName() string {
	return model.BitbucketCloudType
}

func (b *BitbucketProvider) TransformToSourceCodeProvider(config map[string]interface{}) map[string]interface{} {
	p := transformToSourceCodeProvider(config)
	p[client.BitbucketCloudProviderFieldRedirectURL] = formBitbucketRedirectURLFromMap(config)
	return p
}

func transformToSourceCodeProvider(config map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	if m, ok := config["metadata"].(map[string]interface{}); ok {
		result["id"] = fmt.Sprintf("%v:%v", m[client.ObjectMetaFieldNamespace], m[client.ObjectMetaFieldName])
	}
	if t := convert.ToString(config[client.SourceCodeProviderFieldType]); t != "" {
		result[client.SourceCodeProviderFieldType] = client.BitbucketCloudProviderType
	}
	if t := convert.ToString(config[projectNameField]); t != "" {
		result["projectId"] = t
	}
	result[client.BitbucketCloudProviderFieldRedirectURL] = formBitbucketRedirectURLFromMap(config)

	return result
}

func (b *BitbucketProvider) GetProviderConfig(projectID string) (interface{}, error) {
	scpConfigObj, err := b.SourceCodeProviderConfigs.ObjectClient().UnstructuredClient().GetNamespaced(projectID, model.BitbucketCloudType, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve BitbucketConfig, error: %v", err)
	}

	u, ok := scpConfigObj.(runtime.Unstructured)
	if !ok {
		return nil, fmt.Errorf("failed to retrieve BitbucketConfig, cannot read k8s Unstructured data")
	}
	storedBitbucketPipelineConfigMap := u.UnstructuredContent()

	storedBitbucketPipelineConfig := &v3.BitbucketCloudPipelineConfig{}
	if err := mapstructure.Decode(storedBitbucketPipelineConfigMap, storedBitbucketPipelineConfig); err != nil {
		return nil, fmt.Errorf("failed to decode the config, error: %v", err)
	}

	metadataMap, ok := storedBitbucketPipelineConfigMap["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to retrieve BitbucketConfig metadata, cannot read k8s Unstructured data")
	}

	typemeta := &metav1.ObjectMeta{}
	//time.Time cannot decode directly
	delete(metadataMap, "creationTimestamp")
	if err := mapstructure.Decode(metadataMap, typemeta); err != nil {
		return nil, fmt.Errorf("failed to decode the config, error: %v", err)
	}
	storedBitbucketPipelineConfig.ObjectMeta = *typemeta
	storedBitbucketPipelineConfig.APIVersion = "project.cattle.io/v3"
	storedBitbucketPipelineConfig.Kind = v3.SourceCodeProviderConfigGroupVersionKind.Kind
	return storedBitbucketPipelineConfig, nil
}
