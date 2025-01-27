package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if !f.Generate || len(f.Services) == 0 {
				continue
			}
			generateFile(gen, f)
		}
		return nil
	})
}

type twirpFileData struct {
	GeneratedFileName string
	ProtoName         string
	Imports           []string
	Services          []*protogen.Service
}

func generateFile(gen *protogen.Plugin, file *protogen.File) (*protogen.GeneratedFile, error) {
	// setup data struct and extract values given from protoc
	data := twirpFileData{}
	data.GeneratedFileName = file.GeneratedFilenamePrefix
	nameSplit := strings.Split(file.Proto.GetName(), "/")
	data.ProtoName = strings.TrimSuffix(nameSplit[len(nameSplit)-1], ".proto")
	// data.Imports = file.Proto.Dependency
	data.Imports = make([]string, file.Desc.Imports().Len())
	for i := 0; i < file.Desc.Imports().Len(); i++ {
		data.Imports[i] = strings.ReplaceAll(
			file.Desc.Imports().Get(i).Path(),
			".proto", "",
		)
	}
	data.Services = make([]*protogen.Service, len(file.Services))
	copy(data.Services, file.Services)

	// parse template
	protoTemplate, err := template.New(data.ProtoName).Funcs(funcMap).Parse(twirpTemplate)
	if err != nil {
		gen.Error(fmt.Errorf("error parsing template: %v", err))
		return nil, err
	}

	// setup the GenerateFile and execute the template
	filename := file.GeneratedFilenamePrefix + ".pbtwirp.dart"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	err = protoTemplate.Execute(g, &data)
	if err != nil {
		gen.Error(fmt.Errorf("error executing template: %v", err))
		return nil, err
	}

	return g, nil
}

func getGenerateFilePath(f *protogen.File) string {
	split := strings.Split(f.GeneratedFilenamePrefix, "/")
	name := split[len(split)-1]
	return *f.Proto.Package + string(os.PathSeparator) + name
}

func lowerFirstLetter(s string) string {
	return strings.ToLower(string(s[0])) + s[1:]
}

func upperFirstLetter(s string) string {
	return strings.ToUpper(string(s[0])) + s[1:]
}

func removeNewLine(s string) string {
	return strings.ReplaceAll(s, "\n", "")
}

var funcMap = map[string]interface{}{
	"lowerFirstLetter": lowerFirstLetter,
	"upperFirstLetter": upperFirstLetter,
	"removeNewLine":    removeNewLine,
}

const twirpTemplate = `// Code generated by protoc-gen-tarp. DO NOT EDIT. {{ .GeneratedFileName }}

import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:protobuf/protobuf.dart';
import 'package:tart/tart.dart' as twirp;
import '{{ .ProtoName }}.pb.dart';
{{ range $import := .Imports -}}
import '../../../{{ $import }}.pb.dart';
{{end }}

{{ range $service := .Services -}}
{{ removeNewLine $service.Comments.Leading.String }}
abstract class {{ $service.GoName }} {
{{- range $method := $service.Methods }}
  {{ removeNewLine $method.Comments.Leading.String }}
  Future<{{ $method.Output.Desc.Name }}> {{ lowerFirstLetter $method.GoName }}(
		twirp.Context ctx, 
		{{ $method.Input.Desc.Name }} req,
	);
{{- end }}
}
{{- end }}

{{- $protoName := .ProtoName }}

{{ range $service := .Services -}}
{{ removeNewLine $service.Comments.Leading.String }}
class {{ $service.GoName }}JSONClient implements {{ $service.GoName }} {
  String baseUrl;
  String prefix;
  late twirp.ClientHooks hooks;
  late twirp.Interceptor interceptor;

  {{ $service.GoName }}JSONClient(this.baseUrl, this.prefix, {twirp.ClientHooks? hooks, twirp.Interceptor? interceptor}) {
    if (!baseUrl.endsWith('/')) baseUrl += '/';
    if (!prefix.endsWith('/')) prefix += '/';
    if (prefix.startsWith('/')) prefix = prefix.substring(1);

    this.hooks = hooks ?? twirp.ClientHooks();
    this.interceptor = interceptor ?? twirp.chainInterceptor([]);
  }
  {{- range $method := $service.Methods }}

  @override
  Future<{{ $method.Output.Desc.Name }}> {{ lowerFirstLetter $method.GoName }}(
		twirp.Context ctx, 
		{{ $method.Input.Desc.Name }} req,
		) async {
    ctx = twirp.withPackageName(ctx, '{{ $method.Desc.ParentFile.Package.Name }}');
    ctx = twirp.withServiceName(ctx, '{{ $service.GoName }}');
    ctx = twirp.withMethodName(ctx, '{{ $method.GoName }}');
    return interceptor((ctx, req) {
      return call{{ $method.GoName }}(ctx, req);
    })(ctx, req);
  }

  Future<{{ $method.Output.Desc.Name }}> call{{ $method.GoName }}(twirp.Context ctx, {{ $method.Input.Desc.Name }} req) async {
    try {
      Uri url = Uri.parse(baseUrl + prefix + '{{ $method.Parent.Desc.ParentFile.Package }}.{{ $service.GoName }}/{{ $method.GoName }}');
      final data = await doJSONRequest(ctx, url, hooks, req);
      final {{ $method.Output.Desc.Name }} res = {{ $method.Output.Desc.Name }}.create();
      res.mergeFromProto3Json(json.decode(data));
      return Future.value(res);
    } catch (e) {
      rethrow;
    }
  }
  {{- end }}
}

{{ removeNewLine $service.Comments.Leading.String }}
class {{ $service.GoName }}ProtobufClient implements {{ $service.GoName }} {
  String baseUrl;
  String prefix;
  late twirp.ClientHooks hooks;
  late twirp.Interceptor interceptor;

  {{ $service.GoName }}ProtobufClient(this.baseUrl, this.prefix, {twirp.ClientHooks? hooks, twirp.Interceptor? interceptor}) {
    if (!baseUrl.endsWith('/')) baseUrl += '/';
    if (!prefix.endsWith('/')) prefix += '/';
    if (prefix.startsWith('/')) prefix = prefix.substring(1);

    this.hooks = hooks ?? twirp.ClientHooks();
    this.interceptor = interceptor ?? twirp.chainInterceptor([]);
  }
  {{- range $method := $service.Methods }}

  @override
  Future<{{ $method.Output.Desc.Name }}> {{ lowerFirstLetter $method.GoName }}(twirp.Context ctx, {{ $method.Input.Desc.Name }} req) async {
    ctx = twirp.withPackageName(ctx, '{{ $method.Desc.ParentFile.Package.Name }}');
    ctx = twirp.withServiceName(ctx, '{{ $service.GoName }}');
    ctx = twirp.withMethodName(ctx, '{{ $method.GoName }}');
    return interceptor((ctx, req) {
      return call{{ $method.GoName }}(ctx, req);
    })(ctx, req);
  }

  Future<{{ $method.Output.Desc.Name }}> call{{ $method.GoName }}(twirp.Context ctx, {{ $method.Input.Desc.Name }} req) async {
    try {
      Uri url = Uri.parse(baseUrl + prefix + '{{ $method.Parent.Desc.ParentFile.Package }}.{{ $service.GoName }}/{{ $method.GoName }}');
      final data = await doProtobufRequest(ctx, url, hooks, req);
      final {{ $method.Output.Desc.Name }} res = {{ $method.Output.Desc.Name }}.create();
      res.mergeFromBuffer(data);
      return Future.value(res);
    } catch (e) {
      rethrow;
    }
  }
  {{- end }}
}
{{- end }}

Future<List<int>> doProtobufRequest(twirp.Context ctx, Uri url,
    twirp.ClientHooks hooks, GeneratedMessage msgReq) async {
  // setup http client
  final httpClient = http.Client();

  try {
    // create http request
    final req = createRequest(url, ctx, 'application/protobuf');

    // add request data to body
    req.bodyBytes = msgReq.writeToBuffer();

    // call onRequestPrepared hook for user to modify request
    ctx = hooks.onRequestPrepared(ctx, req);

    // send data
    final res = await httpClient.send(req);

    // if success, parse and return response
    if (res.statusCode == 200) {
      List<int> data = <int>[];
      await res.stream.listen((value) {
        data.addAll(value);
      }).asFuture();
      hooks.onResponseReceived(ctx);
      return Future.value(data);
    }

    // we received a twirp related error
    throw twirp.TwirpError.fromJson(
        json.decode(await res.stream.transform(utf8.decoder).join()), ctx);
  } on twirp.TwirpError catch (twirpErr) {
    hooks.onError(ctx, twirpErr);
    rethrow;
  } catch (e) {
    // catch http connection error or from onRequestPrepared
    final twirpErr = twirp.TwirpError.fromConnectionError(e.toString(), ctx);
    hooks.onError(ctx, twirpErr);
    throw twirpErr;
  } finally {
    httpClient.close();
  }
}

Future<String> doJSONRequest(twirp.Context ctx, Uri url,
    twirp.ClientHooks hooks, GeneratedMessage msgReq) async {
  // setup http client
  final httpClient = http.Client();

  try {
    // create http request
    final req = createRequest(url, ctx, 'application/json');

    // add request data to body
    req.body = json.encode(msgReq.toProto3Json());

    // call onRequestPrepared hook for user to modify request
    ctx = hooks.onRequestPrepared(ctx, req);

    // send data
    final res = await httpClient.send(req);

    // if success, parse and return response
    if (res.statusCode == 200) {
      final data = await res.stream.transform(utf8.decoder).join().then((data) {
        hooks.onResponseReceived(ctx);
        return data;
      });
      return Future.value(data);
    }

    // we received a twirp related error
    throw twirp.TwirpError.fromJson(
        json.decode(await res.stream.transform(utf8.decoder).join()), ctx);
  } on twirp.TwirpError catch (twirpErr) {
    hooks.onError(ctx, twirpErr);
    rethrow;
  } catch (e) {
    // catch http connection error or from onRequestPrepared
    final twirpErr = twirp.TwirpError.fromConnectionError(e.toString(), ctx);
    hooks.onError(ctx, twirpErr);
    throw twirpErr;
  } finally {
    httpClient.close();
  }
}

http.Request createRequest(
    Uri url, twirp.Context ctx, String applicationHeader) {
  // setup request
  final req = http.Request("POST", url);

  // add headers from context
  final headersFromCtx = twirp.retrieveHttpRequestHeaders(ctx) ?? {};
  req.headers.addAll(headersFromCtx);

  // add required headers
  req.headers['Accept'] = applicationHeader;
  req.headers['Content-Type'] = applicationHeader;

  return req;
}
`
