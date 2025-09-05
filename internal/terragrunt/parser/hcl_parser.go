package parser

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// TerragruntConfig represents a parsed terragrunt.hcl file
type TerragruntConfig struct {
	TerraformSource   string                       `json:"terraform_source"`
	Inputs            map[string]interface{}       `json:"inputs,omitempty"`
	Dependencies      []Dependency                 `json:"dependencies,omitempty"`
	DependencyBlocks  []DependencyBlock            `json:"dependency_blocks,omitempty"`
	RemoteState       *RemoteStateConfig           `json:"remote_state,omitempty"`
	Locals            map[string]interface{}       `json:"locals,omitempty"`
	Include           []IncludeConfig              `json:"include,omitempty"`
	Generate          map[string]GenerateConfig    `json:"generate,omitempty"`
	Hooks             map[string]HookConfig        `json:"hooks,omitempty"`
	Skip              bool                         `json:"skip,omitempty"`
	IamRole           string                       `json:"iam_role,omitempty"`
	TerraformVersionConstraint string              `json:"terraform_version_constraint,omitempty"`
	FilePath          string                       `json:"file_path"`
	WorkingDir        string                       `json:"working_dir"`
}

// Dependency represents a module dependency
type Dependency struct {
	Name       string `json:"name"`
	ConfigPath string `json:"config_path"`
	Enabled    bool   `json:"enabled"`
}

// DependencyBlock represents a dependency block in terragrunt
type DependencyBlock struct {
	Name            string   `json:"name"`
	ConfigPath      string   `json:"config_path"`
	MockOutputs     map[string]interface{} `json:"mock_outputs,omitempty"`
	MockOutputsMerge bool    `json:"mock_outputs_merge,omitempty"`
	Skip            bool     `json:"skip,omitempty"`
}

// RemoteStateConfig represents remote state configuration
type RemoteStateConfig struct {
	Backend      string                 `json:"backend"`
	Generate     *GenerateConfig        `json:"generate,omitempty"`
	Config       map[string]interface{} `json:"config"`
	DisableInit  bool                   `json:"disable_init,omitempty"`
	DisableDependencyOptimization bool `json:"disable_dependency_optimization,omitempty"`
}

// IncludeConfig represents an include block
type IncludeConfig struct {
	Name         string   `json:"name,omitempty"`
	Path         string   `json:"path"`
	Expose       bool     `json:"expose,omitempty"`
	MergeStrategy string  `json:"merge_strategy,omitempty"`
}

// GenerateConfig represents a generate block
type GenerateConfig struct {
	Path              string `json:"path"`
	IfExists          string `json:"if_exists,omitempty"`
	Contents          string `json:"contents"`
	DisableSignature  bool   `json:"disable_signature,omitempty"`
}

// HookConfig represents a hook configuration
type HookConfig struct {
	Commands       []string `json:"commands"`
	Execute        []string `json:"execute"`
	WorkingDir     string   `json:"working_dir,omitempty"`
	RunOnError     bool     `json:"run_on_error,omitempty"`
}

// Parser handles parsing of Terragrunt HCL files
type Parser struct {
	parser *hclparse.Parser
}

// NewParser creates a new Terragrunt HCL parser
func NewParser() *Parser {
	return &Parser{
		parser: hclparse.NewParser(),
	}
}

// ParseFile parses a terragrunt.hcl file
func (p *Parser) ParseFile(filePath string) (*TerragruntConfig, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return p.ParseContent(content, filePath)
}

// ParseContent parses terragrunt HCL content
func (p *Parser) ParseContent(content []byte, filePath string) (*TerragruntConfig, error) {
	file, diags := p.parser.ParseHCL(content, filePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	config := &TerragruntConfig{
		FilePath:   filePath,
		WorkingDir: filepath.Dir(filePath),
		Inputs:     make(map[string]interface{}),
		Locals:     make(map[string]interface{}),
		Generate:   make(map[string]GenerateConfig),
		Hooks:      make(map[string]HookConfig),
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type")
	}

	// Parse blocks
	for _, block := range body.Blocks {
		switch block.Type {
		case "terraform":
			if err := p.parseTerraformBlock(block, config); err != nil {
				return nil, err
			}
		case "remote_state":
			if err := p.parseRemoteStateBlock(block, config); err != nil {
				return nil, err
			}
		case "include":
			if err := p.parseIncludeBlock(block, config); err != nil {
				return nil, err
			}
		case "dependency":
			if err := p.parseDependencyBlock(block, config); err != nil {
				return nil, err
			}
		case "dependencies":
			if err := p.parseDependenciesBlock(block, config); err != nil {
				return nil, err
			}
		case "generate":
			if err := p.parseGenerateBlock(block, config); err != nil {
				return nil, err
			}
		case "locals":
			if err := p.parseLocalsBlock(block, config); err != nil {
				return nil, err
			}
		}
	}

	// Parse attributes
	for name, attr := range body.Attributes {
		switch name {
		case "terraform_source":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				config.TerraformSource = val.AsString()
			}
		case "iam_role":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				config.IamRole = val.AsString()
			}
		case "skip":
			val, err := p.evalExpression(attr.Expr)
			if err == nil && val.Type() == cty.Bool {
				config.Skip = val.True()
			}
		case "inputs":
			if err := p.parseInputsAttribute(attr, config); err != nil {
				return nil, err
			}
		case "terraform_version_constraint":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				config.TerraformVersionConstraint = val.AsString()
			}
		}
	}

	return config, nil
}

// parseTerraformBlock parses a terraform block
func (p *Parser) parseTerraformBlock(block *hclsyntax.Block, config *TerragruntConfig) error {
	body := block.Body
	for name, attr := range body.Attributes {
		if name == "source" {
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				config.TerraformSource = val.AsString()
			}
		}
	}
	return nil
}

// parseRemoteStateBlock parses a remote_state block
func (p *Parser) parseRemoteStateBlock(block *hclsyntax.Block, config *TerragruntConfig) error {
	if config.RemoteState == nil {
		config.RemoteState = &RemoteStateConfig{
			Config: make(map[string]interface{}),
		}
	}

	body := block.Body
	for name, attr := range body.Attributes {
		switch name {
		case "backend":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				config.RemoteState.Backend = val.AsString()
			}
		case "config":
			configVal, err := p.parseMapExpression(attr.Expr)
			if err == nil {
				config.RemoteState.Config = configVal
			}
		case "disable_init":
			val, err := p.evalExpression(attr.Expr)
			if err == nil && val.Type() == cty.Bool {
				config.RemoteState.DisableInit = val.True()
			}
		}
	}

	// Parse nested generate block
	for _, nestedBlock := range body.Blocks {
		if nestedBlock.Type == "generate" {
			gen := &GenerateConfig{}
			for name, attr := range nestedBlock.Body.Attributes {
				switch name {
				case "path":
					val, _ := p.evalExpression(attr.Expr)
					gen.Path = val.AsString()
				case "if_exists":
					val, _ := p.evalExpression(attr.Expr)
					gen.IfExists = val.AsString()
				}
			}
			config.RemoteState.Generate = gen
		}
	}

	return nil
}

// parseIncludeBlock parses an include block
func (p *Parser) parseIncludeBlock(block *hclsyntax.Block, config *TerragruntConfig) error {
	include := IncludeConfig{}
	
	// Include blocks can have labels (names)
	if len(block.Labels) > 0 {
		include.Name = block.Labels[0]
	}

	body := block.Body
	for name, attr := range body.Attributes {
		switch name {
		case "path":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				include.Path = val.AsString()
			}
		case "expose":
			val, err := p.evalExpression(attr.Expr)
			if err == nil && val.Type() == cty.Bool {
				include.Expose = val.True()
			}
		case "merge_strategy":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				include.MergeStrategy = val.AsString()
			}
		}
	}

	config.Include = append(config.Include, include)
	return nil
}

// parseDependencyBlock parses a dependency block
func (p *Parser) parseDependencyBlock(block *hclsyntax.Block, config *TerragruntConfig) error {
	dep := DependencyBlock{}
	
	// Dependency blocks have labels (names)
	if len(block.Labels) > 0 {
		dep.Name = block.Labels[0]
	}

	body := block.Body
	for name, attr := range body.Attributes {
		switch name {
		case "config_path":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				dep.ConfigPath = val.AsString()
			}
		case "mock_outputs":
			mockOutputs, err := p.parseMapExpression(attr.Expr)
			if err == nil {
				dep.MockOutputs = mockOutputs
			}
		case "mock_outputs_merge_with_state":
			val, err := p.evalExpression(attr.Expr)
			if err == nil && val.Type() == cty.Bool {
				dep.MockOutputsMerge = val.True()
			}
		case "skip":
			val, err := p.evalExpression(attr.Expr)
			if err == nil && val.Type() == cty.Bool {
				dep.Skip = val.True()
			}
		}
	}

	config.DependencyBlocks = append(config.DependencyBlocks, dep)
	return nil
}

// parseDependenciesBlock parses a dependencies block
func (p *Parser) parseDependenciesBlock(block *hclsyntax.Block, config *TerragruntConfig) error {
	body := block.Body
	for name, attr := range body.Attributes {
		if name == "paths" {
			// Parse list of paths
			paths, err := p.parseListExpression(attr.Expr)
			if err == nil {
				for _, path := range paths {
					config.Dependencies = append(config.Dependencies, Dependency{
						ConfigPath: path,
						Enabled:    true,
					})
				}
			}
		}
	}
	return nil
}

// parseGenerateBlock parses a generate block
func (p *Parser) parseGenerateBlock(block *hclsyntax.Block, config *TerragruntConfig) error {
	gen := GenerateConfig{}
	
	// Generate blocks have labels (names)
	var genName string
	if len(block.Labels) > 0 {
		genName = block.Labels[0]
	}

	body := block.Body
	for name, attr := range body.Attributes {
		switch name {
		case "path":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				gen.Path = val.AsString()
			}
		case "if_exists":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				gen.IfExists = val.AsString()
			}
		case "contents":
			val, err := p.evalExpression(attr.Expr)
			if err == nil {
				gen.Contents = val.AsString()
			}
		}
	}

	if genName != "" {
		config.Generate[genName] = gen
	}
	return nil
}

// parseLocalsBlock parses a locals block
func (p *Parser) parseLocalsBlock(block *hclsyntax.Block, config *TerragruntConfig) error {
	body := block.Body
	for name, attr := range body.Attributes {
		val, err := p.evalExpression(attr.Expr)
		if err == nil {
			config.Locals[name] = p.ctyToInterface(val)
		}
	}
	return nil
}

// parseInputsAttribute parses the inputs attribute
func (p *Parser) parseInputsAttribute(attr *hclsyntax.Attribute, config *TerragruntConfig) error {
	inputs, err := p.parseMapExpression(attr.Expr)
	if err != nil {
		return err
	}
	config.Inputs = inputs
	return nil
}

// evalExpression evaluates an HCL expression
func (p *Parser) evalExpression(expr hcl.Expression) (cty.Value, error) {
	// Simple evaluation context
	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{},
		Functions: p.getTerragruntFunctions(),
	}

	val, diags := expr.Value(ctx)
	if diags.HasErrors() {
		return cty.NilVal, fmt.Errorf("failed to evaluate expression: %s", diags.Error())
	}

	return val, nil
}

// parseMapExpression parses a map expression
func (p *Parser) parseMapExpression(expr hcl.Expression) (map[string]interface{}, error) {
	val, err := p.evalExpression(expr)
	if err != nil {
		// If evaluation fails, try to parse as object
		if objExpr, ok := expr.(*hclsyntax.ObjectConsExpr); ok {
			result := make(map[string]interface{})
			for _, item := range objExpr.Items {
				keyVal, _ := item.KeyExpr.Value(nil)
				if keyVal.Type() == cty.String {
					key := keyVal.AsString()
					itemVal, _ := p.evalExpression(item.ValueExpr)
					result[key] = p.ctyToInterface(itemVal)
				}
			}
			return result, nil
		}
		return nil, err
	}

	return p.ctyToMap(val), nil
}

// parseListExpression parses a list expression
func (p *Parser) parseListExpression(expr hcl.Expression) ([]string, error) {
	val, err := p.evalExpression(expr)
	if err != nil {
		return nil, err
	}

	if !val.Type().IsTupleType() && !val.Type().IsListType() {
		return nil, fmt.Errorf("expected list, got %s", val.Type().FriendlyName())
	}

	var result []string
	for it := val.ElementIterator(); it.Next(); {
		_, elem := it.Element()
		if elem.Type() == cty.String {
			result = append(result, elem.AsString())
		}
	}

	return result, nil
}

// ctyToInterface converts a cty.Value to interface{}
func (p *Parser) ctyToInterface(val cty.Value) interface{} {
	if val.IsNull() {
		return nil
	}

	switch {
	case val.Type() == cty.String:
		return val.AsString()
	case val.Type() == cty.Number:
		f, _ := val.AsBigFloat().Float64()
		return f
	case val.Type() == cty.Bool:
		return val.True()
	case val.Type().IsListType() || val.Type().IsTupleType():
		var result []interface{}
		for it := val.ElementIterator(); it.Next(); {
			_, elem := it.Element()
			result = append(result, p.ctyToInterface(elem))
		}
		return result
	case val.Type().IsMapType() || val.Type().IsObjectType():
		return p.ctyToMap(val)
	default:
		return val.GoString()
	}
}

// ctyToMap converts a cty.Value to map[string]interface{}
func (p *Parser) ctyToMap(val cty.Value) map[string]interface{} {
	result := make(map[string]interface{})
	
	if val.Type().IsMapType() || val.Type().IsObjectType() {
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			if k.Type() == cty.String {
				result[k.AsString()] = p.ctyToInterface(v)
			}
		}
	}
	
	return result
}

// getTerragruntFunctions returns Terragrunt-specific functions
func (p *Parser) getTerragruntFunctions() map[string]function.Function {
	// This would include functions like find_in_parent_folders, get_env, etc.
	// For now, return empty map - full implementation would include all Terragrunt functions
	return make(map[string]function.Function)
}

// FindTerragruntFiles finds all terragrunt.hcl files in a directory tree
func FindTerragruntFiles(rootDir string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && strings.HasSuffix(path, "terragrunt.hcl") {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, err
}

// ParseDirectory parses all terragrunt.hcl files in a directory
func (p *Parser) ParseDirectory(dir string) ([]*TerragruntConfig, error) {
	files, err := FindTerragruntFiles(dir)
	if err != nil {
		return nil, err
	}
	
	var configs []*TerragruntConfig
	for _, file := range files {
		config, err := p.ParseFile(file)
		if err != nil {
			// Log error but continue parsing other files
			fmt.Printf("Warning: failed to parse %s: %v\n", file, err)
			continue
		}
		configs = append(configs, config)
	}
	
	return configs, nil
}