package azuredevops

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/johanforsgren/lgtmfaster/internal/domain"
	"github.com/johanforsgren/lgtmfaster/internal/logger"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"gopkg.in/yaml.v3"
)

const (
	gitRefPrefix                = "refs/"
	definitionYamlFilenameField = "yamlFilename"
	yamlPathSeparator           = "/"
)

// yamlParameterTypes maps the run-time parameter types that take a single typed-in value. The
// others (object, step, job, stage, ...) cannot be entered in a form, so the run keeps their default.
var yamlParameterTypes = map[string]domain.PipelineParameterType{
	"":                  domain.PipelineParameterString,
	"string":            domain.PipelineParameterString,
	"number":            domain.PipelineParameterNumber,
	"boolean":           domain.PipelineParameterBoolean,
	"environment":       domain.PipelineParameterString,
	"filePath":          domain.PipelineParameterString,
	"secureFile":        domain.PipelineParameterString,
	"serviceConnection": domain.PipelineParameterString,
}

type yamlPipelineFile struct {
	Parameters yaml.Node `yaml:"parameters"`
}

type yamlPipelineParameter struct {
	Name        string    `yaml:"name"`
	DisplayName string    `yaml:"displayName"`
	Type        string    `yaml:"type"`
	Default     yaml.Node `yaml:"default"`
	Values      []string  `yaml:"values"`
}

// GetPipelineTriggerOptions reads the parameters from the pipeline's YAML file on the default
// branch; a run on another branch is started with that branch's file, whose parameters may differ.
// When the file cannot be read the pipeline can still be started, with its defaults.
func (p *Provider) GetPipelineTriggerOptions(ctx context.Context, identifier domain.PipelineIdentifier) (*domain.PipelineTriggerOptions, error) {
	definition, err := p.client.GetBuildDefinition(ctx, identifier.Project, identifier.PipelineID)
	if err != nil {
		logger.LogError("AZDO_GET_TRIGGER_OPTIONS", fmt.Sprintf("%s/%d", identifier.Project, identifier.PipelineID), err)
		return nil, err
	}
	if definition.Repository == nil || definition.Repository.Id == nil {
		return nil, fmt.Errorf("pipeline %d in project '%s' has no repository", identifier.PipelineID, identifier.Project)
	}
	repoID := *definition.Repository.Id

	refs, err := p.client.ListBranchRefs(ctx, identifier.Project, repoID)
	if err != nil {
		logger.LogError("AZDO_GET_TRIGGER_OPTIONS", fmt.Sprintf("%s/%d", identifier.Project, identifier.PipelineID), err)
		return nil, err
	}
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, extractBranchName(ref.Name))
	}
	sort.Strings(names)
	defaultBranch := extractBranchName(definition.Repository.DefaultBranch)

	options := &domain.PipelineTriggerOptions{
		Branches: domain.RepositoryBranches{Default: defaultBranch, Names: names},
	}
	yamlFilename := definitionYamlFilename(definition)
	if yamlFilename == "" {
		return options, nil
	}
	content, err := p.client.GetFileContentAtBranch(ctx, identifier.Project, repoID, yamlPathSeparator+strings.TrimPrefix(yamlFilename, yamlPathSeparator), defaultBranch)
	if err != nil {
		logger.LogError("AZDO_READ_PIPELINE_YAML", yamlFilename, err)
		return options, nil
	}
	if options.Parameters, err = parsePipelineParameters(content); err != nil {
		logger.LogError("AZDO_PARSE_PIPELINE_YAML", yamlFilename, err)
	}
	return options, nil
}

func (p *Provider) TriggerPipeline(ctx context.Context, trigger domain.PipelineTrigger) (*domain.PipelineRun, error) {
	identifier := trigger.Pipeline
	queue := build.Build{SourceBranch: branchRef(trigger.Branch)}
	if len(trigger.Parameters) > 0 {
		parameters := trigger.Parameters
		queue.TemplateParameters = &parameters
	}

	logger.Log("AzureDevOps: Triggering pipeline %d in %s on %s", identifier.PipelineID, identifier.Project, trigger.Branch)
	queued, err := p.client.QueueBuild(ctx, identifier.Project, identifier.PipelineID, queue)
	if err != nil {
		logger.LogError("AZDO_TRIGGER_PIPELINE", fmt.Sprintf("%s/%d", identifier.Project, identifier.PipelineID), err)
		return nil, err
	}
	run := p.convertPipelineRuns(ctx, []build.Build{*queued}, identifier.Project)[0]
	return &run, nil
}

// branchRef accepts a branch name or a full ref.
func branchRef(branch string) *string {
	ref := branch
	if !strings.HasPrefix(branch, gitRefPrefix) {
		ref = branchRefPrefix + branch
	}
	return &ref
}

// definitionYamlFilename is "" for classic (designer) pipelines, whose process has no YAML file.
func definitionYamlFilename(definition *build.BuildDefinition) string {
	process, ok := definition.Process.(map[string]interface{})
	if !ok {
		return ""
	}
	filename, _ := process[definitionYamlFilenameField].(string)
	return filename
}

// parsePipelineParameters reads the run-time parameters of a pipeline file, leaving out those that
// cannot be typed in.
func parsePipelineParameters(content string) ([]domain.PipelineParameter, error) {
	var file yamlPipelineFile
	if err := yaml.Unmarshal([]byte(content), &file); err != nil {
		return nil, fmt.Errorf("invalid pipeline YAML: %w", err)
	}
	if file.Parameters.Kind != yaml.SequenceNode {
		return nil, nil
	}

	var declared []yamlPipelineParameter
	if err := file.Parameters.Decode(&declared); err != nil {
		return nil, fmt.Errorf("invalid pipeline parameters: %w", err)
	}

	parameters := make([]domain.PipelineParameter, 0, len(declared))
	for _, parameter := range declared {
		parameterType, typedIn := yamlParameterTypes[parameter.Type]
		isScalarDefault := parameter.Default.Kind == 0 || parameter.Default.Kind == yaml.ScalarNode
		if parameter.Name == "" || !typedIn || !isScalarDefault {
			continue
		}
		parameters = append(parameters, domain.PipelineParameter{
			Name:        parameter.Name,
			DisplayName: parameter.DisplayName,
			Type:        parameterType,
			Default:     parameter.Default.Value,
			Values:      parameter.Values,
		})
	}
	return parameters, nil
}
