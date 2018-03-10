package main

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/hashicorp/hcl"
	"io/ioutil"
	"path/filepath"
)

type TerraformResource struct {
	Id         string
	Type       string
	Properties interface{}
	Filename   string
}

func loadHCL(filename string, log LoggingFunction) []interface{} {
	template, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var v interface{}
	err = hcl.Unmarshal([]byte(template), &v)
	if err != nil {
		panic(err)
	}
	jsonData, err := json.MarshalIndent(v, "", "  ")
	log(string(jsonData))

	var hclData interface{}
	err = yaml.Unmarshal(jsonData, &hclData)
	if err != nil {
		panic(err)
	}
	m := hclData.(map[string]interface{})
	results := make([]interface{}, 0)
	for _, key := range []string{"resource", "data"} {
		if m[key] != nil {
			log(fmt.Sprintf("Adding %s", key))
			results = append(results, m[key].([]interface{})...)
		}
	}
	return results
}

func loadTerraformResources(filename string, log LoggingFunction) []TerraformResource {
	hclResources := loadHCL(filename, log)

	resources := make([]TerraformResource, 0)
	for _, resource := range hclResources {
		for resourceType, templateResources := range resource.(map[string]interface{}) {
			if templateResources != nil {
				for _, templateResource := range templateResources.([]interface{}) {
					for resourceId, resource := range templateResource.(map[string]interface{}) {
						tr := TerraformResource{
							Id:         resourceId,
							Type:       resourceType,
							Properties: resource.([]interface{})[0],
							Filename:   filename,
						}
						resources = append(resources, tr)
					}
				}
			}
		}
	}
	return resources
}

func loadTerraformRules(filename string) string {
	terraformRules, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return string(terraformRules)
}

func filterTerraformResourcesByType(resources []TerraformResource, resourceType string) []TerraformResource {
	filtered := make([]TerraformResource, 0)
	for _, resource := range resources {
		if resource.Type == resourceType {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

func validateTerraformResources(resources []TerraformResource, rules []Rule, tags []string, log LoggingFunction) []ValidationResult {
	results := make([]ValidationResult, 0)
	for _, rule := range filterRulesByTag(rules, tags) {
		log(fmt.Sprintf("Rule %s: %s", rule.Id, rule.Message))
		for _, filter := range rule.Filters {
			for _, resource := range filterTerraformResourcesByType(resources, rule.Resource) {
				log(fmt.Sprintf("Checking resource %s", resource.Id))
				status := applyFilter(rule, filter, resource, log)
				if status != "OK" {
					results = append(results, ValidationResult{
						RuleId:       rule.Id,
						ResourceId:   resource.Id,
						ResourceType: resource.Type,
						Status:       status,
						Message:      rule.Message,
						Filename:     resource.Filename,
					})
				}
			}
		}
	}
	return results
}

func shouldIncludeFile(patterns []string, filename string) bool {
	for _, pattern := range patterns {
		_, file := filepath.Split(filename)
		matched, err := filepath.Match(pattern, file)
		if err != nil {
			panic(err)
		}
		if matched {
			return true
		}
	}
	return false
}

func terraform(filename string, rulesFilename string, tags []string, ruleIds []string, log LoggingFunction) {
	resources := loadTerraformResources(filename, log)
	// TODO move the parsing up one level - no need to parse the rules for every single file!
	ruleSet := MustParseRules(loadTerraformRules(rulesFilename))
	if shouldIncludeFile(ruleSet.Files, filename) {
		rules := filterRulesById(ruleSet.Rules, ruleIds)
		results := validateTerraformResources(resources, rules, tags, log)
		printResults(results)
	}
}
