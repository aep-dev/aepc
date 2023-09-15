package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"

	"github.com/spf13/cobra"
	"github.com/toumorokoshi/aep-sandbox/aepc/reader"
	"github.com/toumorokoshi/aep-sandbox/aepc/schema"
	"github.com/toumorokoshi/aep-sandbox/aepc/writer/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

func NewCommand() *cobra.Command {
	var inputFile string
	var outputFile string

	c := &cobra.Command{
		Use:   "aepc",
		Short: "aepc compiles resource representations to full proto rpcs",
		Long:  "aepc compiles resource representations to full proto rpcs",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: error handling
			s := &schema.Service{}
			input, err := readFile(inputFile)
			fmt.Printf("input: %s\n", string(input))
			if err != nil {
				log.Fatalf("unable to read file: %v", err)
			}
			ext := filepath.Ext(inputFile)
			err = unmarshal(ext, input, s)
			if err != nil {
				log.Fatal(err)
			}

			proto, _ := proto.WriteServiceToProto(s)

			err = writeFile(outputFile, proto)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("output file: %s\n", outputFile)
			fmt.Printf("output proto: %s\n", proto)
		},
	}
	c.Flags().StringVarP(&inputFile, "input", "i", "", "input files with resource")
	c.Flags().StringVarP(&outputFile, "output", "o", "", "output file to use")
	return c
}

func unmarshal(ext string, b []byte, s *schema.Service) error {
	switch ext {
	case ".proto":
		if err := reader.ReadServiceFromProto(b, s); err != nil {
			return fmt.Errorf("unable to decode proto %q: %w", string(b), err)
		}
	case ".yaml":
		asJson, err := yaml.YAMLToJSON(b)
		if err != nil {
			return fmt.Errorf("unable to decode yaml to JSON %q: %w", string(b), err)
		}
		if err := protojson.Unmarshal(asJson, s); err != nil {
			log.Fatal(fmt.Errorf("unable to decode proto %q: %w", string(b), err))
		}
	case ".json":
		if err := protojson.Unmarshal(b, s); err != nil {
			return fmt.Errorf("unable to decode json %q: %w", string(b), err)
		}
	default:
		return fmt.Errorf("extension %v is unsupported", ext)
	}
	return nil
}

func readFile(fileName string) ([]byte, error) {
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

func writeFile(fileName string, value []byte) error {
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
