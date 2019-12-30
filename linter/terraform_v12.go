package linter

import (
	"fmt"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"

	//"github.com/ghodss/yaml"
	"github.com/stelligent/config-lint/assertion"

	"github.com/hashicorp/hcl/v2"
	//hclsyntax "github.com/hashicorp/hcl/v2/hclsyntax"
)

type (
	// Terraform12ResourceLoader converts Terraform configuration files into JSON objects
	Terraform12ResourceLoader struct{}

	// Terraform12LoadResult collects all the returns value for parsing an HCL string
	Terraform12LoadResult struct {
		Resources []interface{}
		Data      []interface{}
		Providers []interface{}
		Modules   []interface{}
		Variables []Variable
		AST       *hcl.File
	}
)

// Load parses an HCLv2 file into a collection or Resource objects
func (l Terraform12ResourceLoader) Load(filename string) (FileResources, error) {
	loaded := FileResources{
		Resources: []assertion.Resource{},
	}
	result, err := loadHCLv2(filename)
	if err != nil {
		return loaded, err
	}
	//TODO: MN -- Need to iterate over range here to append all? Seems like I'm missing a GoLang idiom
	for _, element := range result.Resources {
		loaded.Resources = append(loaded.Resources, element.(assertion.Resource))
	}

	assertion.DebugJSON("loaded.Resources", loaded.Resources)

	return loaded, nil
}

func loadHCLv2(filename string) (Terraform12LoadResult, error) {
	result := Terraform12LoadResult{
		Resources: []interface{}{},
		Data:      []interface{}{},
		Providers: []interface{}{},
		Modules:   []interface{}{},
		Variables: []Variable{},
	}

	var file *hcl.File

	parser := hclparse.NewParser()
	file, _ = parser.ParseHCLFile(filename)
	_, diags := file.Body.Content(terraformSchema)
	if diags != nil {
		fmt.Printf("ERROR:\n %v\n", diags)
		return result, diags
	}

	// New custom parser
	hcl2Parser := New()
	hcl2FileBlocks, err := hcl2Parser.parseFile(file)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return result, err
	}

	var Vars map[string]cty.Value

	// Returns (Blocks, *hcl.EvalContext)
	hcl2Blocks, _ := hcl2Parser.buildEvaluationContext(hcl2FileBlocks, filename, Vars, true)

	hcl2Resources := hcl2Blocks.OfType("resource")
	for _, block := range hcl2Resources {
		hcl2Resource := createResourceStruct(block, filename)
		result.Resources = append(result.Resources, hcl2Resource)
	}

	hcl2Variables := hcl2Blocks.OfType("variable")
	for _,block := range hcl2Variables{
		hcl2Var := createVariableStruct(block)
		result.Variables = append(result.Variables, hcl2Var)
	}

	//assertion.Debugf("LoadHCL Variables: %v\n", result.Variables)
	return result, nil
}

func createVariableStruct(hcl2Variable *Block) Variable{
	variable := Variable{
		Name : hcl2Variable.Labels()[0],
	}
	props := make(map[string]interface{})
	variable.Value = props

	attributes := hcl2Variable.GetAttributes()
	for _,b := range attributes{
		attName :=  b.hclAttribute.Name
		attValue := b.Value().AsString()
		props[attName] = attValue
	}
	return variable
}

func createResourceStruct(hcl2Resource *Block, filename string) assertion.Resource{
	resource := assertion.Resource{
		Category:   hcl2Resource.Type(),
		Type:       hcl2Resource.Labels()[0],
		ID:         hcl2Resource.Labels()[1],
		Filename:   filename,
		LineNumber: hcl2Resource.Range().StartLine,
	}
	props := make(map[string]interface{})
	// nestedProps := make(map[string]interface{})
	resource.Properties = props
	atts := hcl2Resource.GetAttributes()
	for _, b := range atts {
		if b.Type().IsObjectType(){
			attName := b.Name()
			attValue := b.Value().AsValueMap()
			props[attName] = attValue
		} else{
			attName := b.Name()
			attValue := b.Value().AsString()
			props[attName] = attValue
		}
	}
	return resource
}

// PostLoad resolves variable expressions
func (l Terraform12ResourceLoader) PostLoad(inputResources FileResources) ([]assertion.Resource, error) {
	//for _, resource := range inputResources.Resources {
	//	resource.Properties = tf12ReplaceVariables(resource.Properties, inputResources.Variables)
	//}
	//for _, resource := range inputResources.Resources {
	//	properties, err := tf12ParseJSONDocuments(resource.Properties)
	//	if err != nil {
	//		return inputResources.Resources, err
	//	}
	//	resource.Properties = properties
	//}
	return inputResources.Resources, nil
}
