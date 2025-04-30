// Copyright 2023 Yusuke Fredrick Tsutsumi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"

	"github.com/aep-dev/aep-lib-go/pkg/api"
	"github.com/aep-dev/aep-lib-go/pkg/proto"
	"github.com/aep-dev/aepc/validator"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	var inputFile string
	var outputFilePrefix string

	c := &cobra.Command{
		Use:   "aepc",
		Short: "aepc compiles resource representations to full proto rpcs",
		Long:  "aepc compiles resource representations to full proto rpcs",
		Run: func(cmd *cobra.Command, args []string) {
			err := ProcessInput(inputFile, outputFilePrefix)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
	c.Flags().StringVarP(&inputFile, "input", "i", "", "input files with resource")
	c.Flags().StringVarP(&outputFilePrefix, "output", "o", "", "output file to write to. File types will be appended to this prefix)")
	return c
}

func ProcessInput(inputFile, outputFilePrefix string) error {
	outputDir := filepath.Dir(outputFilePrefix)
	input, err := ReadFile(inputFile)
	fmt.Printf("input: %s\n", string(input))
	if err != nil {
		return fmt.Errorf("unable to read file: %w", err)
	}
	ext := filepath.Ext(inputFile)
	a, err := deserializeAPI(ext, input)
	if err != nil {
		return fmt.Errorf("unable to unmarshal file: %w", err)
	}
	errors := validator.ValidateAPI(a)
	if len(errors) > 0 {
		return fmt.Errorf("error validating service: %v", errors)
	}
	proto, err := proto.APIToProtoString(a, outputDir)
	if err != nil {
		return fmt.Errorf("error writing service proto: %w", err)
	}
	protoFile := fmt.Sprintf("%s.proto", outputFilePrefix)
	err = WriteFile(protoFile, proto)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}
	fmt.Printf("output proto file: %s\n", protoFile)
	openapi, err := a.ConvertToOpenAPIBytes()
	if err != nil {
		return fmt.Errorf("error building openapi: %w", err)
	}
	openapiFile := fmt.Sprintf("%s_openapi.json", outputFilePrefix)
	err = WriteFile(openapiFile, openapi)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}
	fmt.Printf("output openapi file: %s\n", openapiFile)
	yamlOpenAPI, err := yaml.JSONToYAML(openapi)
	if err != nil {
		return fmt.Errorf("error converting openapi json to yaml: %w", err)
	}
	yamlOpenAPIFile := fmt.Sprintf("%s_openapi.yaml", outputFilePrefix)
	err = WriteFile(yamlOpenAPIFile, yamlOpenAPI)
	if err != nil {
		return fmt.Errorf("error writing yaml file: %w", err)
	}
	fmt.Printf("output openapi yaml file: %s\n", yamlOpenAPIFile)
	return nil
}

func deserializeAPI(ext string, b []byte) (*api.API, error) {
	switch ext {
	case ".yaml":
		asJson, err := yaml.YAMLToJSON(b)
		if err != nil {
			return nil, fmt.Errorf("unable to decode yaml to JSON %q: %w", string(b), err)
		}
		api, err := api.LoadAPIFromJson(asJson)
		if err != nil {
			log.Fatal(fmt.Errorf("unable to unmarshal json %q: %w", string(b), err))
		}
		return api, nil
	case ".json":
		api, err := api.LoadAPIFromJson(b)
		if err != nil {
			log.Fatal(fmt.Errorf("unable to unmarshal json %q: %w", string(b), err))
		}
		return api, nil
	default:
		return nil, fmt.Errorf("extension %v is unsupported", ext)
	}
}

func ReadFile(fileName string) ([]byte, error) {
	var value []byte
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	bytesRead := 1
	for bytesRead > 0 {
		readBytes := make([]byte, 10000)
		bytesRead, err = f.Read(readBytes)
		if bytesRead > 0 {
			value = append(value, readBytes[:bytesRead]...)
		}
		if err != io.EOF && err != nil {
			return nil, err
		}
	}
	err = f.Close()
	if err != nil {
		return nil, err
	}
	return value, nil
}

func WriteFile(fileName string, value []byte) error {
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(value)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}
