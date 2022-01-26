package generator

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"text/template"

	tstypes "github.com/go-generalize/go-easyparser/types"
	"golang.org/x/xerrors"
)

type Generator struct {
	types map[string]tstypes.Type
	generatorParam

	converted   map[string]string
	prereserved map[string]string
	reserved    map[string]struct{}

	ExternalImporter func(*tstypes.Object) *ExternalImporter
	Converter        func(v tstypes.Type, meta *metadata) *ConvertedType
	imported         map[string]struct{}
}

type ExternalImporter struct {
	Path string
	Name string
}

type objectEntry struct {
	Converter string
	Field     string
	JsonField string
	Type      string
	Tag       string
	Default   string
	Required  bool
}

type object struct {
	Name   string
	Fields []objectEntry
}

type constantEnum struct {
	Name  string
	Value string
}

type constant struct {
	Name  string
	Base  string
	Enums []constantEnum
}

type generatorParam struct {
	Consts   []constant
	Objects  []object
	Imported []string

	UseTimePackage bool
}

type metadata struct {
	upperStructName string
	inlineIndex     int
}

func NewGenerator(types map[string]tstypes.Type, prereserved []string) *Generator {
	prs := map[string]string{}
	for _, p := range prereserved {
		_, name := SplitPackegeStruct(p)
		prs[name] = p
	}

	return &Generator{
		types:          types,
		generatorParam: generatorParam{},
		converted:      map[string]string{},
		reserved:       map[string]struct{}{},
		prereserved:    prs,
		imported:       map[string]struct{}{},
	}
}

// ConvertedType is a result type for Converter
type ConvertedType struct {
	Default     string
	Required    bool
	Converter   string
	Type        string
	Base        string
	ImportAlias string
}

func (g *Generator) convert(v tstypes.Type, meta *metadata) ConvertedType {
	if conv := g.Converter(v, meta); conv != nil {
		return *conv
	}

	switch v := v.(type) {
	case *tstypes.Array:
		ct := g.convert(v.Inner, meta)
		if ct.Converter != "" {
			ct.Converter = fmt.Sprintf("ListConverter<%s, %s>(%s)", ct.Type, ct.Base, ct.Converter)
		}
		ct.Type = "List<" + ct.Type + ">"
		ct.Base = "List<" + ct.Base + ">"

		return ct
	case *tstypes.Object:
		return g.convertObject(v, meta)
	case *tstypes.String:
		return g.convertString(v, meta)
	case *tstypes.Number:
		return g.convertNumber(v, meta)
	case *tstypes.Boolean:
		return ConvertedType{
			Type:      "bool",
			Base:      "bool",
			Converter: "DoNothingConverter<bool>()",
			Default:   "false",
		}
	case *tstypes.Date:
		g.UseTimePackage = true
		return ConvertedType{
			Default:   "null",
			Type:      "DateTime?",
			Base:      "String",
			Converter: "DateTimeConverter()",
		}
	case *tstypes.Nullable:
		ct := g.convert(v.Inner, meta)

		if strings.HasSuffix(ct.Type, "?") {
			return ct
		}

		return ConvertedType{
			Default:   "null",
			Converter: fmt.Sprintf("NullableConverter<%s, %s>(%s)", ct.Type, ct.Base, ct.Converter),
			Type:      ct.Type + "?",
			Base:      ct.Base + "?",
		}
	case *tstypes.Any:
		return ConvertedType{
			Type:      "dynamic",
			Base:      "dynamic",
			Converter: "DoNothingConverter<dynamic>()",
			Default:   "null",
		}
	case *tstypes.Map:
		key, value := g.convert(v.Key, meta), g.convert(v.Value, meta)
		ct := ConvertedType{
			Type:    "Map<" + key.Type + ", " + value.Type + ">",
			Default: "const {}",
		}

		if value.Converter != "" {
			ct.Converter = fmt.Sprintf("MapConverter<%s, %s, %s>(%s)", key.Type, value.Type, value.Base, value.Converter)
			ct.Base = "Map<" + key.Type + ", " + value.Base + ">"
		}

		return ct
	default:
		panic("unsupported")
	}
}

func (g *Generator) convertString(str *tstypes.String, upper *metadata) ConvertedType {
	if len(str.Enum) == 0 {
		return ConvertedType{
			Type:      "String",
			Base:      "String",
			Converter: "DoNothingConverter<String>()",
			Default:   `''`,
		}
	}

	if name, ok := g.converted[str.Name]; ok {
		return ConvertedType{
			Default:   name + "." + convertEnumKey(str.RawEnum[0].Key),
			Converter: name + "Converter()",
			Base:      "String",
			Type:      name,
		}
	}

	name := g.getConvertedType(str.Name, upper)
	consts := make([]constantEnum, 0, len(str.RawEnum))

	for _, e := range str.RawEnum {
		consts = append(consts, constantEnum{
			Name:  convertEnumKey(e.Key),
			Value: "'" + e.Value + "'",
		})
	}

	g.Consts = append(g.Consts, constant{
		Name:  name,
		Base:  "String",
		Enums: consts,
	})

	return ConvertedType{
		Default:   name + "." + convertEnumKey(str.RawEnum[0].Key),
		Converter: name + "Converter()",
		Base:      "String",
		Type:      name,
	}
}

func (g *Generator) convertNumber(num *tstypes.Number, upper *metadata) ConvertedType {
	if len(num.Enum) == 0 {
		t := getBasicTypeName(num.RawType)
		return ConvertedType{
			Default:   "0",
			Type:      t,
			Base:      t,
			Converter: fmt.Sprintf("DoNothingConverter<%s>()", t),
		}
	}

	baseType := getBasicTypeName(num.RawType)
	if name, ok := g.converted[num.Name]; ok {
		return ConvertedType{
			Default:   name + "." + convertEnumKey(num.RawEnum[0].Key),
			Converter: name + "Converter()",
			Base:      baseType,
			Type:      name,
		}
	}

	name := g.getConvertedType(num.Name, upper)

	enums := make([]constantEnum, 0, len(num.RawEnum))

	for _, e := range num.RawEnum {
		enums = append(enums, constantEnum{
			Name:  convertEnumKey(e.Key),
			Value: fmt.Sprint(e.Value),
		}) // Support multiple types
	}

	g.Consts = append(g.Consts, constant{
		Name:  name,
		Base:  baseType,
		Enums: enums,
	})

	return ConvertedType{
		Default:   name + "." + convertEnumKey(num.RawEnum[0].Key),
		Converter: name + "Converter()",
		Base:      baseType,
		Type:      name,
	}
}

func (g *Generator) getImportAlias(path string) string {
	cs := sha256.Sum256([]byte(path))

	return "external_" + hex.EncodeToString(cs[:])[:7]
}

func (g *Generator) convertObject(obj *tstypes.Object, upper *metadata) ConvertedType {
	var converted object

	if g.ExternalImporter != nil {
		ei := g.ExternalImporter(obj)

		if ei != nil {
			g.imported[ei.Path] = struct{}{}

			return ConvertedType{
				Required:    true,
				Converter:   ei.Name + "Converter()",
				Base:        "Map<String, dynamic>",
				Type:        ei.Name,
				ImportAlias: g.getImportAlias(ei.Path),
			}
		}
	}

	if name, ok := g.converted[obj.Name]; ok {
		return ConvertedType{
			Required:  true,
			Converter: name + "Converter()",
			Base:      "Map<String, dynamic>",
			Type:      name,
		}
	}

	name := g.getConvertedType(obj.Name, upper)

	type objectEntryWithKey struct {
		tstypes.ObjectEntry
		name string
	}

	entries := make([]objectEntryWithKey, 0, len(obj.Entries))
	for k, v := range obj.Entries {
		entries = append(entries, objectEntryWithKey{
			ObjectEntry: v,
			name:        k,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RawName < entries[j].RawName
	})

	for i, e := range entries {
		if reflect.StructTag(e.RawTag).Get("json") == "-" {
			continue
		}

		t := e.Type

		if e.Optional {
			t = &tstypes.Nullable{Inner: t}
		}

		ct := g.convert(t, &metadata{upperStructName: name, inlineIndex: i})

		importPrefix := ""
		if ct.ImportAlias != "" {
			importPrefix = ct.ImportAlias + "."
		}

		converted.Fields = append(converted.Fields, objectEntry{
			Field:     ReplaceFieldName(e.ObjectEntry.RawName),
			JsonField: e.name,
			Converter: importPrefix + ct.Converter,
			Type:      importPrefix + ct.Type,
			Tag:       e.RawTag,
			Default:   ct.Default,
			Required:  ct.Required,
		})
	}

	converted.Name = name
	g.Objects = append(g.Objects, converted)

	return ConvertedType{
		Required:  true,
		Converter: name + "Converter()",
		Base:      "Map<String, dynamic>",
		Type:      name,
	}
}

func (g *Generator) getConvertedType(fullName string, meta *metadata) string {
	var name string

	if fullName == "" {
		name = meta.upperStructName + "Inline" + fmt.Sprintf("%03d", meta.inlineIndex)
	} else {
		_, name = SplitPackegeStruct(fullName)

		prev, prereserved := g.prereserved[fullName]
		_, reserved := g.reserved[name]
		if (prereserved && prev != fullName) || reserved {
			hash := fmt.Sprintf("%x", sha1.Sum([]byte(fullName)))

			name = name + "_" + hash[:4]
		}

		g.reserved[name] = struct{}{}
	}
	g.converted[fullName] = name

	return name
}

//go:embed templates/template.dart
var templateBase string

func (g *Generator) Generate() (string, error) {
	for _, v := range g.types {
		g.convert(v, nil)
	}

	sort.Slice(g.Objects, func(i, j int) bool {
		return g.Objects[i].Name < g.Objects[j].Name
	})
	sort.Slice(g.Consts, func(i, j int) bool {
		return g.Consts[i].Name < g.Consts[j].Name
	})
	g.Imported = make([]string, 0, len(g.imported))
	for k := range g.imported {
		g.Imported = append(g.Imported, k)
	}
	sort.Slice(g.Imported, func(i, j int) bool {
		return g.Imported[i] < g.Imported[j]
	})

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"GetImportAlias": g.getImportAlias,
	}).Parse(templateBase))

	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, g.generatorParam); err != nil {
		return "", xerrors.Errorf("failed to generate template: %w", err)
	}

	return buf.String(), nil
}
