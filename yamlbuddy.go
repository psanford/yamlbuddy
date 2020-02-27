package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

var toJSON = flag.Bool("json", false, "Convert to json")
var check = flag.Bool("check", false, "Check syntax")
var inFile = flag.String("in", "", "input file (defaults to stdin)")

func main() {
	flag.Parse()

	args := flag.Args()

	if *inFile == "" && len(args) == 1 {
		inFile = &args[0]
	}

	var (
		input  = os.Stdin
		output = os.Stdout
		err    error
	)

	if *inFile != "" {
		input, err = os.Open(*inFile)
		if err != nil {
			log.Fatalf("Could not open input file: %s", err)
		}
		defer input.Close()
	} else {
		info, err := os.Stdin.Stat()
		if err == nil {
			mode := info.Mode()
			if mode&os.ModeCharDevice != 0 {
				fmt.Fprintf(os.Stderr, "Reading from stdin\n")
			}
		}
	}

	inBody, err := ioutil.ReadAll(input)
	if err != nil {
		log.Fatalf("Error reading input: %s", err)
	}

	if len(inBody) == 0 {
		return
	}

	var parsed interface{}
	err = yaml.Unmarshal(inBody, &parsed)
	if err != nil {
		log.Fatalf("Error parsing: %s", err)
	}

	if *toJSON {
		jsonable, err := fixInterfaceMaps(parsed)
		if err != nil {
			log.Fatalf("Error marshaling json: %s", err)
		}
		out, err := json.MarshalIndent(&jsonable, "", "  ")
		if err != nil {
			log.Fatalf("Error marshaling json: %s", err)
		}

		if _, err := output.Write(out); err != nil {
			log.Fatalf("Error writing result: %s", err)
		}

		output.Write([]byte("\n"))
	}
}

func fixInterfaceMaps(in interface{}) (interface{}, error) {
	out := in
	switch inT := in.(type) {
	case map[interface{}]interface{}:
		jsonObj := make(map[string]interface{})
		for kface, v := range inT {
			k, ok := kface.(string)
			if !ok {
				return nil, fmt.Errorf("cannot convert key %v to string", kface)
			}

			fixedV, err := fixInterfaceMaps(v)
			if err != nil {
				return nil, err
			}
			jsonObj[k] = fixedV
		}
		out = jsonObj
	case []interface{}:
		for i, obj := range inT {
			fixedV, err := fixInterfaceMaps(obj)
			if err != nil {
				return nil, err
			}
			inT[i] = fixedV
		}
	}

	return out, nil
}
