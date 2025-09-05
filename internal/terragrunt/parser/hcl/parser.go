package hcl

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// TerragruntConfig represents a parsed Terragrunt configuration
type TerragruntConfig struct {
	Include      *Include               `json:"include,omitempty"`
	Terraform    *TerraformBlock        `json:"terraform,omitempty"`
	Inputs       map[string]interface{} `json:"inputs,omitempty"`
	Dependencies *Dependencies          `json:"dependencies,omitempty"`
	Locals       map[string]interface{} `json:"locals,omitempty"`
}

// Include represents the include block in Terragrunt
type Include struct {
	Path   string `json:"path,omitempty"`
	Expose bool   `json:"expose,omitempty"`
}

// TerraformBlock represents the terraform block in Terragrunt
type TerraformBlock struct {
	Source  string                 `json:"source"`
	Vars    map[string]interface{} `json:"vars,omitempty"`
	Backend map[string]interface{} `json:"backend,omitempty"`
}

// Dependencies represents the dependencies block in Terragrunt
type Dependencies struct {
	Paths []string `json:"paths"`
}

// Parser parses Terragrunt HCL files
type Parser struct {
	parser *hclparse.Parser
}

// NewParser creates a new Terragrunt parser
func NewParser() *Parser {
	return &Parser{
		parser: hclparse.NewParser(),
	}
}

// ParseFile parses a Terragrunt HCL file
func (p *Parser) ParseFile(filePath string) (*TerragruntConfig, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.Parse(content, filePath)
}

// Parse parses Terragrunt HCL content
func (p *Parser) Parse(content []byte, filename string) (*TerragruntConfig, error) {
	file, diags := p.parser.ParseHCL(content, filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse errors: %s", diags.Error())
	}

	config := &TerragruntConfig{
		Inputs: make(map[string]interface{}),
		Locals: make(map[string]interface{}),
	}

	// Parse the HCL body
	if err := p.parseBody(file.Body, config); err != nil {
		return nil, err
	}

	return config, nil
}

// parseBody parses the HCL body into TerragruntConfig
func (p *Parser) parseBody(body hcl.Body, config *TerragruntConfig) error {
	content, _, diags := body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "include"},
			{Type: "terraform"},
			{Type: "dependencies"},
			{Type: "locals"},
		},
		Attributes: []hcl.AttributeSchema{
			{Name: "inputs"},
		},
	})

	if diags.HasErrors() {
		return fmt.Errorf("failed to decode body: %s", diags.Error())
	}

	// Parse attributes
	for name, attr := range content.Attributes {
		if name == "inputs" {
			val, err := p.extractValue(attr.Expr, nil)
			if err == nil {
				if m, ok := val.(map[string]interface{}); ok {
					config.Inputs = m
				}
			}
		}
	}

	// Parse blocks
	for _, block := range content.Blocks {
		switch block.Type {
		case "include":
			if err := p.parseInclude(block, config); err != nil {
				return err
			}
		case "terraform":
			if err := p.parseTerraform(block, config); err != nil {
				return err
			}
		case "dependencies":
			if err := p.parseDependencies(block, config); err != nil {
				return err
			}
		case "locals":
			if err := p.parseLocals(block, config); err != nil {
				return err
			}
		}
	}

	return nil
}

// parseInclude parses the include block
func (p *Parser) parseInclude(block *hcl.Block, config *TerragruntConfig) error {
	config.Include = &Include{}

	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return fmt.Errorf("failed to decode include: %s", diags.Error())
	}

	for name, attr := range attrs {
		val, err := p.extractValue(attr.Expr, nil)
		if err != nil {
			continue
		}

		switch name {
		case "path":
			if s, ok := val.(string); ok {
				config.Include.Path = s
			}
		case "expose":
			if b, ok := val.(bool); ok {
				config.Include.Expose = b
			}
		}
	}

	return nil
}

// parseTerraform parses the terraform block
func (p *Parser) parseTerraform(block *hcl.Block, config *TerragruntConfig) error {
	config.Terraform = &TerraformBlock{
		Vars:    make(map[string]interface{}),
		Backend: make(map[string]interface{}),
	}

	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return fmt.Errorf("failed to decode terraform: %s", diags.Error())
	}

	for name, attr := range attrs {
		val, err := p.extractValue(attr.Expr, nil)
		if err != nil {
			continue
		}

		switch name {
		case "source":
			if s, ok := val.(string); ok {
				config.Terraform.Source = s
			}
		case "extra_arguments", "vars":
			if m, ok := val.(map[string]interface{}); ok {
				config.Terraform.Vars = m
			}
		}
	}

	return nil
}

// parseDependencies parses the dependencies block
func (p *Parser) parseDependencies(block *hcl.Block, config *TerragruntConfig) error {
	config.Dependencies = &Dependencies{
		Paths: []string{},
	}

	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return fmt.Errorf("failed to decode dependencies: %s", diags.Error())
	}

	for name, attr := range attrs {
		if name == "paths" {
			val, err := p.extractValue(attr.Expr, nil)
			if err != nil {
				continue
			}

			if list, ok := val.([]interface{}); ok {
				for _, item := range list {
					if s, ok := item.(string); ok {
						config.Dependencies.Paths = append(config.Dependencies.Paths, s)
					}
				}
			}
		}
	}

	return nil
}

// parseLocals parses the locals block
func (p *Parser) parseLocals(block *hcl.Block, config *TerragruntConfig) error {
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return fmt.Errorf("failed to decode locals: %s", diags.Error())
	}

	for name, attr := range attrs {
		val, err := p.extractValue(attr.Expr, nil)
		if err != nil {
			continue
		}
		config.Locals[name] = val
	}

	return nil
}

// extractValue extracts a value from an HCL expression
func (p *Parser) extractValue(expr hcl.Expression, ctx *hcl.EvalContext) (interface{}, error) {
	if ctx == nil {
		ctx = &hcl.EvalContext{
			Functions: hclFunctions(),
		}
	}

	val, diags := expr.Value(ctx)
	if diags.HasErrors() {
		// For now, return a simple error
		return nil, fmt.Errorf("failed to evaluate expression: %s", diags.Error())
	}

	return p.ctyToGo(val)
}

// ctyToGo converts a cty.Value to a Go value
func (p *Parser) ctyToGo(val cty.Value) (interface{}, error) {
	if val.IsNull() {
		return nil, nil
	}

	ty := val.Type()
	switch {
	case ty == cty.String:
		return val.AsString(), nil
	case ty == cty.Number:
		bf := val.AsBigFloat()
		f, _ := bf.Float64()
		return f, nil
	case ty == cty.Bool:
		return val.True(), nil
	case ty.IsListType() || ty.IsTupleType():
		var result []interface{}
		for it := val.ElementIterator(); it.Next(); {
			_, elem := it.Element()
			goVal, err := p.ctyToGo(elem)
			if err != nil {
				return nil, err
			}
			result = append(result, goVal)
		}
		return result, nil
	case ty.IsMapType() || ty.IsObjectType():
		result := make(map[string]interface{})
		for it := val.ElementIterator(); it.Next(); {
			key, elem := it.Element()
			keyStr := key.AsString()
			goVal, err := p.ctyToGo(elem)
			if err != nil {
				return nil, err
			}
			result[keyStr] = goVal
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", ty.FriendlyName())
	}
}

// hclFunctions returns the standard Terragrunt functions
func hclFunctions() map[string]function.Function {
	return map[string]function.Function{
		"find_in_parent_folders": function.New(&function.Spec{
			Params: []function.Parameter{},
			Type:   function.StaticReturnType(cty.String),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				// Mock implementation
				return cty.StringVal("../terragrunt.hcl"), nil
			},
		}),
		"path_relative_to_include": function.New(&function.Spec{
			Params: []function.Parameter{},
			Type:   function.StaticReturnType(cty.String),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				// Mock implementation
				return cty.StringVal("."), nil
			},
		}),
		"get_env": function.New(&function.Spec{
			Params: []function.Parameter{
				{Name: "name", Type: cty.String},
				{Name: "default", Type: cty.String},
			},
			Type: function.StaticReturnType(cty.String),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				// Mock implementation
				if len(args) > 1 {
					return args[1], nil
				}
				return cty.StringVal(""), nil
			},
		}),
	}
}

// ParseDirectory parses all Terragrunt files in a directory
func (p *Parser) ParseDirectory(dir string) ([]*TerragruntConfig, error) {
	pattern := filepath.Join(dir, "**", "terragrunt.hcl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob files: %w", err)
	}

	var configs []*TerragruntConfig
	for _, file := range files {
		config, err := p.ParseFile(file)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Warning: failed to parse %s: %v\n", file, err)
			continue
		}
		configs = append(configs, config)
	}

	return configs, nil
}
