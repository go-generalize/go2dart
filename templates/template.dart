import 'dart:convert';

import 'package:intl/intl.dart';

abstract class JsonConverter<T, S> {
  const JsonConverter();

  T fromJson(dynamic json);
  S toJson(T object);
}

class ListConverter<T, Base> implements JsonConverter<List<T>, List<Base>> {
  const ListConverter(this.internalConverter);

  final JsonConverter<T, Base> internalConverter;

  @override 
  List<T> fromJson(dynamic arr) {
    return List<dynamic>.from(arr).map((e) => internalConverter.fromJson(e)).toList();
  }

  @override
  List<Base> toJson(List<T> arr) {
    return arr.map((e) => internalConverter.toJson(e) as Base).toList();
  }
}

class MapConverter<K, T, Base> implements JsonConverter<Map<K, T>, Map<K, Base>> {
  const MapConverter(this.internalConverter);

  final JsonConverter<T, Base> internalConverter;

  @override 
  Map<K, T> fromJson(dynamic m) {
    return Map<K, dynamic>.from(m).map((key, value) => MapEntry<K, T>(key, internalConverter.fromJson(value)));
  }

  @override
  Map<K, Base> toJson(Map<K, T> m) {
    return m.map((key, value) => MapEntry<K, Base>(key, internalConverter.toJson(value)));
  }
}

class DateTimeConverter implements JsonConverter<DateTime, String> {
  const DateTimeConverter();

  @override 
  DateTime fromJson(dynamic s) {
    return DateTime.parse(s as String);
  }

  @override
  String toJson(DateTime? dt) {
    return (dt ?? DateTime.fromMillisecondsSinceEpoch(0, isUtc: true)).toUtc().toIso8601String();
  }
}

class NullableConverter<T, Base> implements JsonConverter<T?, Base?> {
  const NullableConverter(this.internalConverter);

  final JsonConverter<T, Base> internalConverter;

  @override 
  T? fromJson(dynamic s) {
    return s == null ? null : internalConverter.fromJson(s);
  }

  @override
  Base? toJson(T? dt) {
    return dt == null ? null : internalConverter.toJson(dt);
  }
}

class DoNothingConverter<T> implements JsonConverter<T, T> {
  const DoNothingConverter();

  @override 
  T fromJson(dynamic s) {
    return s as T;
  }

  @override
  T toJson(T d) {
    return d;
  }
}

{{ range $elm := .Consts }}
enum {{ $elm.Name }} {
{{- range $c := $elm.Enums }}
    {{ $c.Name }},
{{- end }}
}

class {{ $elm.Name }}Converter implements JsonConverter<{{ $elm.Name }}, {{ $elm.Base }}> {
  const {{ $elm.Name }}Converter();

  @override 
  {{ $elm.Name }} fromJson(dynamic s) {
    return {{ $elm.Name }}Extension.fromJson(s as {{ $elm.Base }});
  }

  @override
  {{ $elm.Base }} toJson({{ $elm.Name }} s) {
    return s.toJson();
  }
}

extension {{ $elm.Name }}Extension on {{ $elm.Name }} {
  static final enumToValue = {
{{- range $c := $elm.Enums }}
    {{ $elm.Name }}.{{ $c.Name }}: {{ $c.Value }},
{{- end }}
  };
  static final valueToEnum = {
{{- range $c := $elm.Enums }}
    {{ $c.Value }}: {{ $elm.Name }}.{{ $c.Name }},
{{- end }}
  };

  static {{ $elm.Name }} fromJson(dynamic d) {
    return {{ $elm.Name }}Extension.valueToEnum[d]!;
  }

  {{ $elm.Base }} toJson() {
    return {{ $elm.Name }}Extension.enumToValue[this]!;
  }
}
{{ end -}}

{{- range $elm := .Objects }}
class {{ $elm.Name }}Converter implements JsonConverter<{{ $elm.Name }}, Map<String, dynamic>> {
  const {{ $elm.Name }}Converter();

  @override 
  {{ $elm.Name }} fromJson(dynamic s) {
    return {{ $elm.Name }}.fromJson(Map<String, dynamic>.from(s));
  }

  @override
  Map<String, dynamic> toJson({{ $elm.Name }} s) {
    return s.toJson();
  }
}

class {{ $elm.Name }} {
{{- range $f := $elm.Fields }}
  {{ $f.Type }} {{ $f.Field }};
{{- end }}


  {{ $elm.Name }}({
{{- range $f := $elm.Fields }}
    {{ if $f.Required}}required {{end}}this.{{ $f.Field }}{{ if and (ne $f.Default "null") (ne $f.Default "") }} = {{$f.Default}}{{ end }},
{{- end }}
  });

  factory {{ $elm.Name }}.fromJson(Map<String, dynamic> json) {
    return {{ $elm.Name }}(
{{- range $f := $elm.Fields }}
      {{ $f.Field }}: {{ if ne $f.Converter "" }}{{ $f.Converter }}.fromJson(json['{{ $f.JsonField }}']){{ else }}json['{{ $f.JsonField }}'] as {{ $f.Type }}{{end}},
{{- end }}
    );
  }

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
{{- range $f := $elm.Fields }}
      '{{ $f.JsonField }}': {{ if ne $f.Converter "" }}{{ $f.Converter }}.toJson({{ $f.Field }}){{ else }}{{ $f.Field }}{{end}},
{{- end }}
    };
  }
}
{{ end }}
