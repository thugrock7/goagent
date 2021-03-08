package main

import (
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/tallstoat/pbparser"
)

func getGoPackageName(pf pbparser.ProtoFile) string {
	for _, opt := range pf.Options {
		if opt.Name == "go_package" {
			return opt.Value
		}
	}
	return pf.PackageName
}

var zeroValues = map[string]string{
	"string": `""`,
	"bool":   `false`,
}

func toPublicFieldName(name string) string {
	return strings.Title(name)
}

const filepathSeparator = string(filepath.Separator)

type protobufImportModuleProvider struct {
	dir string
}

func (pi *protobufImportModuleProvider) Provide(module string) (io.Reader, error) {
	modulePath := pi.dir + filepathSeparator + module
	if strings.HasPrefix(module, "google/protobuf/") {
		modulePath = pi.dir + filepathSeparator + "protobuf" + filepathSeparator + "src" + filepathSeparator + module
	}

	raw, err := ioutil.ReadFile(modulePath)
	if err != nil {
		return nil, err
	}

	r := strings.NewReader(string(raw[:]))
	return r, nil
}

func isEnum(pf pbparser.ProtoFile, name string) bool {
	for _, e := range pf.Enums {
		if e.Name == name {
			return true
		}
	}

	return false
}

func shouldSkipOtherLanguageAgent(typeName string) bool {
	return strings.HasSuffix(typeName, "Agent") && !strings.HasPrefix(typeName, "Go")
}

func main() {
	var outDir = flag.String("o", ".", "OUT_DIR for the generated code.")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println(`Usage: generator PROTO_FILE
Parse PROTO_FILE and generate output value objects`)
		return
	}

	filename := flag.Arg(0)
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Unable to open the proto file %q: %v", filename, err)
		os.Exit(1)
	}

	path, err := os.Getwd()
	if err != nil {
		fmt.Printf("Unable to get current working directory: %v", err)
		os.Exit(1)
	}

	pf, err := pbparser.Parse(f, &protobufImportModuleProvider{
		path + filepathSeparator + "cmd" + filepathSeparator + "generator",
	})
	if err != nil {
		fmt.Printf("Unable to parse proto file %q: %v \n", filename, err)
		os.Exit(1)
	}

	fqpn := getGoPackageName(pf)
	pn := strings.Split(fqpn, "/")
	c := fmt.Sprintf("// Code generated by hypertrace/goagent/config. DO NOT EDIT.\n\n")
	c += fmt.Sprintf("package %s\n\n", pn[len(pn)-1])
	// even if we don't need this import, worth to add it as gofmt is going to remove it.
	c += "import wrappers \"github.com/golang/protobuf/ptypes/wrappers\"\n\n"

	for _, m := range pf.Messages {
		if shouldSkipOtherLanguageAgent(m.Name) {
			continue
		}

		mapFields := []pbparser.FieldElement{}

		c += "// loadFromEnv loads the data from env vars, defaults and makes sure all values are initialized.\n"
		c += fmt.Sprintf("func (x *%s) loadFromEnv(prefix string, defaultValues *%s) {\n", m.Name, m.Name)
		for _, mf := range m.Fields {
			if shouldSkipOtherLanguageAgent(mf.Type.Name()) {
				continue
			}

			if mf.Label == "oneof" {
				// currently we don't have a way to handle oneof labels
				// in env vars.
				continue
			}

			fieldName := toPublicFieldName(strcase.ToCamel(mf.Name))
			envPrefix := strings.ToUpper(strcase.ToSnake(mf.Name))
			fieldType := mf.Type.Name()
			if mf.Label == "repeated" {
				c += fmt.Sprintf(
					"    if rawVals, ok := getArrayStringEnv(prefix + \"%s\"); ok {\n",
					strings.ToUpper(mf.Name),
				)

				if namedType, ok := mf.Type.(pbparser.NamedDataType); ok {
					if isEnum(pf, namedType.Name()) {
						c += fmt.Sprintf(`		vals := []%s{}
						for _, rawVal := range rawVals {
							vals = append(vals, %s(%s_value[rawVal]))
						}
					`, namedType.Name(), namedType.Name(), namedType.Name())
					} else {
						// unsupported
						c += "vals = nil"
					}

					c += fmt.Sprintf("        x.%s = vals\n", fieldName)
				}
				c += fmt.Sprintf("    } else if len(x.%s) == 0 && len(defaultValues.%s) > 0 {\n", fieldName, fieldName)
				c += fmt.Sprintf("        x.%s = defaultValues.%s\n", fieldName, fieldName)
				c += fmt.Sprintf("    }\n\n")
			} else if strings.HasPrefix(fieldType, "google.protobuf.") {
				_type := mf.Type.Name()[16 : len(mf.Type.Name())-5] // 16 = len("google.protobuf.")
				c += fmt.Sprintf(
					"    if val, ok := get%sEnv(prefix + \"%s\"); ok {\n",
					strings.Title(_type),
					envPrefix,
				)
				c += fmt.Sprintf("        x.%s = &wrappers.%sValue{Value: val}\n", fieldName, _type)
				c += fmt.Sprintf("    } else if x.%s == nil {\n", fieldName)
				c += "        // when there is no value to set we still prefer to initialize the variable to avoid\n"
				c += "        // `nil` checks in the consumers.\n"
				c += fmt.Sprintf("        x.%s = new(wrappers.%sValue)\n", fieldName, _type)
				c += fmt.Sprintf("        if defaultValues != nil && defaultValues.%s != nil {\n", fieldName)
				c += fmt.Sprintf("            x.%s = &wrappers.%sValue{Value: defaultValues.%s.Value}\n", fieldName, _type, fieldName)
				c += "        }\n"
				c += "    }\n"
			} else if namedType, ok := mf.Type.(pbparser.NamedDataType); ok {
				if isEnum(pf, namedType.Name()) {
					c += fmt.Sprintf(
						"    if rawVal, ok := getStringEnv(prefix + \"%s\"); ok {\n",
						envPrefix,
					)
					c += fmt.Sprintf("        x.%s = %s(%s_value[rawVal])\n", fieldName, namedType.Name(), namedType.Name())
					c += fmt.Sprintf("    }\n\n")
				} else {
					c += fmt.Sprintf("    if x.%s == nil { x.%s = new(%s) }\n", fieldName, fieldName, namedType.Name())
					c += fmt.Sprintf("    x.%s.loadFromEnv(prefix + \"%s_\", defaultValues.%s)\n", fieldName, envPrefix, fieldName)
				}
			} else if strings.HasPrefix(fieldType, "map") {
				mapFields = append(mapFields, mf)
				c += fmt.Sprintf("    if defaultValues != nil && len(defaultValues.%s) > 0 {\n", fieldName)

				kType, vType := getSubtypesFromMap(mf.Type.Name())
				c += fmt.Sprintf("        if x.%s == nil { x.%s = make(map[%s]%s) } \n", fieldName, fieldName, kType, vType)
				c += fmt.Sprintf("        for k, v := range defaultValues.%s {\n", fieldName)
				c += "            // defaults should not override existing resource attributes unless empty\n"
				c += fmt.Sprintf("            if _, ok := x.%s[k]; !ok {", fieldName)
				c += fmt.Sprintf("                x.%s[k] = v\n", fieldName)
				c += "            }\n"
				c += "        }\n"
				c += "    }\n\n"
			} else {
				c += fmt.Sprintf(
					"    if val, ok := get%sEnv(prefix + \"%s\"); ok {\n",
					strings.Title(fieldType),
					envPrefix,
				)
				c += fmt.Sprintf("        x.%s = val\n", fieldName)
				c += fmt.Sprintf("    } else if x.%s == %s && defaultValues != nil && defaultValues.%s != %s {\n", fieldName, zeroValues[fieldType], fieldName, zeroValues[fieldType])
				c += fmt.Sprintf("        x.%s = defaultValues.%s\n", fieldName, fieldName)
				c += fmt.Sprintf("    }\n\n")
			}
		}
		c += "}\n\n"

		if len(mapFields) > 0 {
			for _, mf := range mapFields {
				addMapFieldSetter(&c, m, mf)
				c += "\n"
			}
		}
	}

	baseFilename := filepath.Base(filename)
	outputFile := baseFilename[0 : len(baseFilename)-6] // 6 = len(".proto")

	bc := []byte(c)
	fbc, err := format.Source(bc)
	if err != nil {
		fmt.Printf("failed to format the content, writing unformatted: %v\n", err)
		fbc = bc
	}

	err = writeToFile(*outDir+"/"+outputFile+".pbloader.go", fbc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getSubtypesFromMap(_type string) (string, string) {
	_type = strings.Replace(_type, " ", "", -1)
	t := strings.Split(_type[4:len(_type)-1], ",")
	if t[0] != "string" || t[1] != "string" {
		// While it is possible things work smooth, we better file and refine this rule
		// for field map setters.
		log.Fatalf("unsupported map subtypes: key %q value %q", t[0], t[1])
	}

	return t[0], t[1]
}

func addMapFieldSetter(c *string, m pbparser.MessageElement, mf pbparser.FieldElement) {
	fieldName := toPublicFieldName(strcase.ToCamel(mf.Name))
	*c += fmt.Sprintf("// Put%s sets values in the %s map.\n", fieldName, fieldName)

	kType, vType := getSubtypesFromMap(mf.Type.Name())

	*c += fmt.Sprintf("func (x *%s) Put%s(m map[%s]%s) {\n", m.Name, fieldName, kType, vType)
	*c += fmt.Sprintf("    if len(m) == 0 { return }\n")
	*c += fmt.Sprintf("    if x.%s == nil { x.%s = make(map[%s]%s) } \n", fieldName, fieldName, kType, vType)
	*c += "    for k, v := range m {\n"
	*c += fmt.Sprintf("            x.%s[k] = v\n", fieldName)
	*c += "    }\n"
	*c += "}\n"
}

func writeToFile(filename string, content []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %v", filename, err)
	}
	defer f.Close()
	_, err = f.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write into file %q: %v", filename, err)
	}

	return nil
}
