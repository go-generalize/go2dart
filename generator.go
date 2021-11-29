package generator

import (
	"bytes"
	"crypto/sha1"
	_ "embed"
	"fmt"
	"sort"
	"text/template"

	tstypes "github.com/go-generalize/go2ts/pkg/types"
	"golang.org/x/xerrors"
)

type Generator struct {
	types map[string]tstypes.Type
	generatorParam

	converted   map[string]string
	prereserved map[string]string
	reserved    map[string]struct{}
}

type objectEntry struct {
	Converter string
	Field     string
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
	Consts  []constant
	Objects []object

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
	}
}

type convertedType struct {
	Default   string
	Required  bool
	Converter string
	Type      string
	Base      string
}

func (g *Generator) convert(v tstypes.Type, meta *metadata) convertedType {
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
		return convertedType{
			Type:      "bool",
			Base:      "bool",
			Converter: "DoNothingConverter<bool>()",
			Default:   "false",
		}
	case *tstypes.Date:
		g.UseTimePackage = true
		return convertedType{
			Default:   "null",
			Type:      "DateTime?",
			Base:      "String",
			Converter: "DateTimeConverter()",
		}
	case *tstypes.Nullable:
		ct := g.convert(v.Inner, meta)

		return convertedType{
			Default:   "null",
			Converter: fmt.Sprintf("NullableConverter<%s, %s>(%s)", ct.Type, ct.Base, ct.Converter),
			Type:      ct.Type + "?",
			Base:      ct.Base + "?",
		}
	case *tstypes.Any:
		return convertedType{Type: "dynamic", Default: "null"}
	case *tstypes.Map:
		key, value := g.convert(v.Key, meta), g.convert(v.Value, meta)
		ct := convertedType{
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

func (g *Generator) convertString(str *tstypes.String, upper *metadata) convertedType {
	if len(str.Enum) == 0 {
		return convertedType{
			Type:      "String",
			Base:      "String",
			Converter: "DoNothingConverter<String>()",
			Default:   `''`,
		}
	}

	if name, ok := g.converted[str.Name]; ok {
		return convertedType{
			Default:   name + "." + str.RawEnum[0].Key,
			Converter: name + "Converter()",
			Base:      "String",
			Type:      name,
		}
	}

	name := g.getConvertedType(str.Name, upper)
	consts := make([]constantEnum, 0, len(str.RawEnum))

	for _, e := range str.RawEnum {
		consts = append(consts, constantEnum{
			Name:  e.Key,
			Value: "'" + e.Value + "'",
		})
	}

	g.Consts = append(g.Consts, constant{
		Name:  name,
		Base:  "String",
		Enums: consts,
	})

	return convertedType{
		Default:   name + "." + str.RawEnum[0].Key,
		Converter: name + "Converter()",
		Base:      "String",
		Type:      name,
	}
}

func (g *Generator) convertNumber(num *tstypes.Number, upper *metadata) convertedType {
	if len(num.Enum) == 0 {
		t := getBasicTypeName(num.RawType)
		return convertedType{
			Default:   "0",
			Type:      t,
			Base:      t,
			Converter: fmt.Sprintf("DoNothingConverter<%s>()", t),
		}
	}

	baseType := getBasicTypeName(num.RawType)
	if name, ok := g.converted[num.Name]; ok {
		return convertedType{
			Default:   name + "." + num.RawEnum[0].Key,
			Converter: name + "Converter()",
			Base:      baseType,
			Type:      name,
		}
	}

	name := g.getConvertedType(num.Name, upper)

	enums := make([]constantEnum, 0, len(num.RawEnum))

	for _, e := range num.RawEnum {
		enums = append(enums, constantEnum{
			Name:  e.Key,
			Value: fmt.Sprint(e.Value),
		}) // Support multiple types
	}

	g.Consts = append(g.Consts, constant{
		Name:  name,
		Base:  baseType,
		Enums: enums,
	})

	return convertedType{
		Default:   name + "." + num.RawEnum[0].Key,
		Converter: name + "Converter()",
		Base:      baseType,
		Type:      name,
	}
}

func (g *Generator) convertObject(obj *tstypes.Object, upper *metadata) convertedType {
	var converted object

	if name, ok := g.converted[obj.Name]; ok {
		return convertedType{
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
		t := e.Type

		if e.Optional {
			t = &tstypes.Nullable{Inner: t}
		}

		ct := g.convert(t, &metadata{upperStructName: name, inlineIndex: i})

		converted.Fields = append(converted.Fields, objectEntry{
			Field:     e.name,
			Converter: ct.Converter,
			Type:      ct.Type,
			Tag:       e.RawTag,
			Default:   ct.Default,
			Required:  ct.Required,
		})
	}

	converted.Name = name
	g.Objects = append(g.Objects, converted)

	return convertedType{
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

	tmpl := template.Must(template.New("").Parse(templateBase))

	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, g.generatorParam); err != nil {
		return "", xerrors.Errorf("failed to generate template: %w", err)
	}

	return buf.String(), nil
}