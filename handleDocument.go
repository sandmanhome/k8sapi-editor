package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type Swagger struct {
	Swagger             string                 `json:"swagger"`
	Info                Info                   `json:"info"`
	SecurityDefinitions SecurityDefinitions    `json:"securityDefinitions"`
	Security            []interface{}          `json:"security"`
	Paths               map[string]interface{} `json:"paths"`
	Definitions         map[string]interface{} `json:"definitions"`
}
type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}
type SecurityDefinitions struct {
	BearerToken BearerToken `json:"BearerToken"`
}
type BearerToken struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	In          string `json:"in"`
}

type Parameter struct {
	UniqueItems bool   `json:"uniqueItems"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Name        string `json:"name"`
	In          string `json:"in"`
	Required    bool   `json:"required"`
	Enum		[]string `json:"enum,omitempty"`
}

type Get struct {
	Consumes    []string            `json:"consumes"`
	Description string              `json:"description"`
	OperationId string              `json:"operationId"`
	Produces    []string            `json:"produces"`
	Parameters 	[]Parameter         `json:"parameters"`
	Responses   map[string]Response `json:"responses"`
	Schemes     []string            `json:"schemes,omitempty"`
	Tags        []string            `json:"tags"`
}

type Response struct {
	Description string  `json:"description"`
	Schema      *Schema `json:"schema,omitempty"`
}
type Schema struct {
	Ref string `json:"$ref"`
}

type Definition struct {
	Description string              `json:"description"`
	Properties  map[string]Property `json:"properties"`
}

type Property struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	Items       *Items `json:"items,omitempty"`
}
type Items struct {
	Type string `json:"type,omitempty"`
	Ref  string `json:"$ref,omitempty"`
}

func generateClustersGetPath() *Get {
	get := &Get{
		Consumes:    []string{"application/json"},
		Description: "get all available k8s clusters",
		OperationId: "",
		Produces:    []string{"application/json"},
		Parameters: []Parameter{
			Parameter{
				UniqueItems: true,
				Type:        "string",
				Description: "providerType of k8s cluster",
				Name:        "providerType",
				In:          "query",
				Required:    false,
				Enum: []string{"private"},
			},
			Parameter{
				UniqueItems: true,
				Type:        "string",
				Description: "regionName of k8s cluster",
				Name:        "regionName",
				In:          "query",
				Required:    false,
				Enum: []string{"logicA"},
			},
		},
		Responses: map[string]Response{
			"200": {
				Description: "OK",
				Schema:      &Schema{Ref: "#/definitions/com.luckincoffee.cloud.pkg.api.v1.Clusters"},
			},
			"401": {
				Description: "Unauthorized",
				Schema:      nil,
			},
		},
		Tags: []string{"core"},
	}
	return get
}

func main() {
	swagger := &Swagger{}
	data, _ := ioutil.ReadFile("swagger-k8s-1.13.1.json")
	json.Unmarshal(data, swagger)
	//apis := len(swagger.Paths)
	//definitions := len(swagger.Definitions)

	//打印已有 path
	fmt.Printf("========= all paths ===========\n")
	for k, _ := range swagger.Paths {
		fmt.Printf("\"%s\"\n", k)
	}

	// 白名单过滤 Path
	var restorePath map[string]interface{} = map[string]interface{}{}
	whiteList := []string{
		"namespaces",
		"nodes",
		"events",
		"pods",
		"deployments",
		"services",
		"ingresses",
		"secrets",
		"configmaps",
		"serviceaccounts",
		"clusterroles",
		"clusterrolebindings",
		"roles",
		"rolebindings",
	}
	for _, whiteKey := range whiteList {
		for k, v := range swagger.Paths {
			if strings.Contains(k, whiteKey) {
				restorePath[k] = v
			}
		}
	}
	swagger.Paths = restorePath

	fmt.Printf("========= new paths===========\n")
	for k, _ := range swagger.Paths {
		fmt.Printf("\"%s\"\n", k)
	}

	var pathDefinitions map[string]interface{} = map[string]interface{}{}
	for _, v := range swagger.Paths {
		input, _ := json.Marshal(v)
		var definitionList []string = []string{}
		extractRefDefinitions(swagger.Definitions, &definitionList, string(input))
		for _, definitionKey := range definitionList {
			pathDefinitions[definitionKey] = swagger.Definitions[definitionKey]
		}
	}

	var restoreDefinitions map[string]interface{} = map[string]interface{}{}
	for k, v := range pathDefinitions {
		restoreDefinitions[k] = v
		input, _ := json.Marshal(v)
		var definitionList []string = []string{}
		extractRefDefinitions(swagger.Definitions, &definitionList, string(input))
		for _, definitionKey := range definitionList {
			restoreDefinitions[definitionKey] = swagger.Definitions[definitionKey]
		}
	}
	swagger.Definitions = restoreDefinitions

	data, _ = json.MarshalIndent(swagger, "", "    ")
	for k, _ := range swagger.Paths {
		v := swagger.Paths[k]
		if _, ok := v.(map[string]interface{})["parameters"]; !ok {
			v.(map[string]interface{})["parameters"] = []interface{}{}
		}
		v.(map[string]interface{})["parameters"] = append(v.(map[string]interface{})["parameters"].([]interface{}), &Parameter{
			UniqueItems: true,
			Type:        "string",
			Description: "id of k8s cluster",
			Name:        "clusterid",
			In:          "query",
			Required:    true,
		})
	}
	//for k, _ := range swagger.Paths {
	//	delete(swagger.Paths, k)
	//}
	//for k, _ := range swagger.Definitions {
	//	delete(swagger.Definitions, k)
	//}
	swagger.Paths["/api/v3/clusters"] = map[string]interface{}{"get": generateClustersGetPath()}
	swagger.Definitions["com.luckincoffee.cloud.pkg.api.v1.Clusters"] = generateClustersDefinitions()
	swagger.Definitions["com.luckincoffee.cloud.pkg.api.v1.Cluster"] = generateClusterDefinitions()
	swagger.Info.Title = "luckincoffee cloud apis"
	swagger.Info.Version = "v1.0"
	data, _ = json.MarshalIndent(swagger, "", "    ")
	ioutil.WriteFile("cloud-api-1.0.json", data, os.ModeType)
}

func extractRefDefinitions(Definitions map[string]interface{}, res *[]string, input string) {
	reg, err := regexp.Compile("\\\"\\$ref\\\":\\s?\\\"(.*?)\\\"")
	if err != nil {
		return
	}

	resultList := reg.FindAllStringSubmatch(input, -1)
	for _, v := range resultList {
		if len(v) > 1 {
			inputKey := strings.Replace(v[1], "#/definitions/", "", 1)
			*res = append(*res, inputKey)
			if obj, ok := Definitions[inputKey]; ok {
				newInput, _ := json.Marshal(obj)
				extractRefDefinitions(Definitions, res, string(newInput))
			}
		}
	}
}

func generateClustersDefinitions() *Definition {
	return &Definition{
		Description: "Clusters List the k8s clusters that are available",
		Properties: map[string]Property{
			"apiVersion": {
				Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values.",
				Type:        "string",
			},
			"items": {
				Description: "",
				Type:        "array",
				Items: &Items{
					Ref: "#definitions/com.luckincoffee.cloud.pkg.api.v1.Cluster",
				},
			},
		},
	}
}

func generateClusterDefinitions() *Definition {
	return &Definition{
		Description: "Cluster is the k8s clusters that are available",
		Properties: map[string]Property{
			"id": {
				Description: "ID is the id of the k8s cluster",
				Type:        "string",
			},
			"name": {
				Description: "NAME is the name of the k8s cluster",
				Type:        "string",
			},
			"providerType": {
				Description: "Provider is the k8s cluster providerType",
				Type:        "string",
			},
		},
	}
}
