package main

const tFile = `// This code was autogenerated from {{.GetName}}, do not edit.

{{- $pkgName := GoPackageName .}}
{{- $pkgSubject := GetPkgSubject .}}
{{- $pkgSubjectPrefix := GetPkgSubjectPrefix .}}
{{- $pkgSubjectParams := GetPkgSubjectParams .}}
package {{$pkgName}}

import (
	"context"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/go-nats"
	{{- if Prometheus}}
	"github.com/prometheus/client_golang/prometheus"
	{{- end}}
	"github.com/rapidloop/nrpc"
)

{{range .Service -}}
// {{.GetName}}Server is the interface that providers of the service
// {{.GetName}} should implement.
type {{.GetName}}Server interface {
	{{- range .Method}}
	{{- $resultType := GetResultType .}}
	{{.GetName}}(ctx context.Context, req {{GetPkg $pkgName .GetInputType}}) (resp {{GetPkg $pkgName $resultType}}, err error)
	{{- end}}
}

{{- if Prometheus}}

var (
	// The request completion time, measured at client-side.
	clientRCTFor{{.GetName}} = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "nrpc_client_request_completion_time_seconds",
			Help:       "The request completion time for {{.GetName}} calls, measured client-side.",
			Objectives: map[float64]float64{0.9: 0.01, 0.95: 0.01, 0.99: 0.001},
			ConstLabels: map[string]string{
				"service": "{{.GetName}}",
			},
		},
		[]string{"method"})

	// The handler execution time, measured at server-side.
	serverHETFor{{.GetName}} = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "nrpc_server_handler_execution_time_seconds",
			Help:       "The handler execution time for {{.GetName}} calls, measured server-side.",
			Objectives: map[float64]float64{0.9: 0.01, 0.95: 0.01, 0.99: 0.001},
			ConstLabels: map[string]string{
				"service": "{{.GetName}}",
			},
		},
		[]string{"method"})

	// The counts of calls made by the client, classified by result type.
	clientCallsFor{{.GetName}} = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nrpc_client_calls_count",
			Help: "The count of calls made by the client.",
			ConstLabels: map[string]string{
				"service": "{{.GetName}}",
			},
		},
		[]string{"method", "encoding", "result_type"})

	// The counts of requests handled by the server, classified by result type.
	serverRequestsFor{{.GetName}} = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nrpc_server_requests_count",
			Help: "The count of requests handled by the server.",
			ConstLabels: map[string]string{
				"service": "{{.GetName}}",
			},
		},
		[]string{"method", "encoding", "result_type"})
)
{{- end}}

// {{.GetName}}Handler provides a NATS subscription handler that can serve a
// subscription using a given {{.GetName}}Server implementation.
type {{.GetName}}Handler struct {
	ctx    context.Context
	nc     nrpc.NatsConn
	server {{.GetName}}Server
}

func New{{.GetName}}Handler(ctx context.Context, nc nrpc.NatsConn, s {{.GetName}}Server) *{{.GetName}}Handler {
	return &{{.GetName}}Handler{
		ctx:    ctx,
		nc:     nc,
		server: s,
	}
}

func (h *{{.GetName}}Handler) Subject() string {
	return "{{$pkgSubjectPrefix}}
	{{- range $pkgSubjectParams -}}
		*.
	{{- end -}}
	{{GetServiceSubject .}}
	{{- range GetServiceSubjectParams . -}}
		.*
	{{- end -}}
	.>"
}

func (h *{{.GetName}}Handler) Handler(msg *nats.Msg) {
	// extract method name & encoding from subject
	{{ if ne 0 (len $pkgSubjectParams)}}pkgParams{{else}}_{{end -}},
	{{- if ne 0 (len (GetServiceSubjectParams .))}} svcParams{{else}} _{{end -}}
	, name, encoding, err := nrpc.ParseSubject(
		"{{$pkgSubject}}", {{len $pkgSubjectParams}}, "{{GetServiceSubject .}}", {{len (GetServiceSubjectParams .)}}, msg.Subject)

	ctx := h.ctx
	{{- range $i, $name := $pkgSubjectParams }}
	ctx = context.WithValue(ctx, "nrpc-pkg-{{$name}}", pkgParams[{{$i}}])
	{{- end }}
	{{- range $i, $name := GetServiceSubjectParams . }}
	ctx = context.WithValue(ctx, "nrpc-svc-{{$name}}", svcParams[{{$i}}])
	{{- end }}
	// call handler and form response
	var resp proto.Message
	var replyError *nrpc.Error
{{- if Prometheus}}
	var elapsed float64
{{- end}}
	switch name {
	{{- $serviceName := .GetName}}{{- range .Method}}
	case "{{.GetName}}":
		var req {{GetPkg $pkgName .GetInputType}}
		if err := nrpc.Unmarshal(encoding, msg.Data, &req); err != nil {
			log.Printf("{{.GetName}}Handler: {{.GetName}} request unmarshal failed: %v", err)
			replyError = &nrpc.Error{
				Type: nrpc.Error_CLIENT,
				Message: "bad request received: " + err.Error(),
			}
{{- if Prometheus}}
			serverRequestsFor{{$serviceName}}.WithLabelValues(
				"{{.GetName}}", encoding, "unmarshal_fail").Inc()
{{- end}}
		} else {
{{- if Prometheus}}
			start := time.Now()
{{- end}}
			{{- if HasFullReply . }}
			resp, replyError = nrpc.CaptureErrors(
				func()(proto.Message, error){
					result, err := h.server.{{.GetName}}(ctx, req)
					if err != nil {
						return nil, err
					}
					return &{{ GetPkg $pkgName .GetOutputType }}{
						&{{ GetPkg $pkgName .GetOutputType }}_Result{
							Result: {{if HasPointerResultType .}}&{{end}}result,
						},
					}, err
				})
			{{- else}}
			resp, replyError = nrpc.CaptureErrors(
				func()(proto.Message, error){
					innerResp, err := h.server.{{.GetName}}(ctx, req)
					if err != nil {
						return nil, err
					}
					return &innerResp, err
				})
			{{- end}}
{{- if Prometheus}}
			elapsed = time.Since(start).Seconds()
{{- end}}
			if replyError != nil {
				log.Printf("{{.GetName}}Handler: {{.GetName}} handler failed: %s", replyError.Error())
{{- if Prometheus}}
				serverRequestsFor{{$serviceName}}.WithLabelValues(
					"{{.GetName}}", encoding, "handler_fail").Inc()
{{- end}}
			}
		}
{{- end}}
	default:
		log.Printf("{{.GetName}}Handler: unknown name %q", name)
		replyError = &nrpc.Error{
			Type: nrpc.Error_CLIENT,
			Message: "unknown name: " + name,
		}
{{- if Prometheus}}
		serverRequestsFor{{.GetName}}.WithLabelValues(
			"{{.GetName}}", encoding, "name_fail").Inc()
{{- end}}
	}

	// encode and send response
	err = nrpc.Publish(resp, replyError, h.nc, msg.Reply, encoding) // error is logged
{{- if Prometheus}}
	if err != nil {
		serverRequestsFor{{$serviceName}}.WithLabelValues(
			name, encoding, "sendreply_fail").Inc()
	} else if replyError == nil {
		serverRequestsFor{{$serviceName}}.WithLabelValues(
			name, encoding, "success").Inc()
	}

	// report metric to Prometheus
	serverHETFor{{$serviceName}}.WithLabelValues(name).Observe(elapsed)
{{- else}}
	if err != nil {
		log.Println("{{.GetName}}Handler: {{.GetName}} handler failed to publish the response: %s", err)
	}
{{- end}}
}

type {{.GetName}}Client struct {
	nc      nrpc.NatsConn
	{{- if ne 0 (len $pkgSubject)}}
	PkgSubject string
	{{- end}}
	{{- range $pkgSubjectParams}}
	PkgParam{{ . }} string
	{{- end}}
	Subject string
	{{- range GetServiceSubjectParams .}}
	SvcParam{{ . }} string
	{{- end}}
	Encoding string
	Timeout time.Duration
}

func New{{.GetName}}Client(nc nrpc.NatsConn
	{{- range $pkgSubjectParams -}}
	, pkgParam{{.}} string
	{{- end -}}
	{{- range GetServiceSubjectParams . -}}
	, svcParam{{ . }} string
	{{- end -}}
	) *{{.GetName}}Client {
	return &{{.GetName}}Client{
		nc:      nc,
		PkgSubject: "{{$pkgSubject}}",
		{{- range $pkgSubjectParams}}
		PkgParam{{.}}: pkgParam{{.}},
		{{- end}}
		Subject: "{{GetServiceSubject .}}",
		{{- range GetServiceSubjectParams .}}
		SvcParam{{.}}: svcParam{{.}},
		{{- end}}
		Encoding: "protobuf",
		Timeout: 5 * time.Second,
	}
}
{{$serviceName := .GetName}}
{{$serviceSubjectParams := GetServiceSubjectParams .}}
{{- range .Method}}
{{- $resultType := GetResultType .}}
func (c *{{$serviceName}}Client) {{.GetName}}(req {{GetPkg $pkgName .GetInputType}}) (resp {{GetPkg $pkgName $resultType}}, err error) {
{{- if Prometheus}}
	start := time.Now()
{{- end}}

	subject := {{ if ne 0 (len $pkgSubject) -}}
		c.PkgSubject + "." + {{end}}
	{{- range $pkgSubjectParams -}}
	    c.PkgParam{{.}} + "." + {{end -}}
	c.Subject + "." + {{range $serviceSubjectParams -}}
	    c.SvcParam{{.}} + "." + {{end -}}
	"{{.GetName}}";

	// call
	{{- if HasFullReply .}}
	var reply {{GetPkg $pkgName .GetOutputType}}
	err = nrpc.Call(&req, &reply, c.nc, subject, c.Encoding, c.Timeout)
	{{- else}}
	err = nrpc.Call(&req, &resp, c.nc, subject, c.Encoding, c.Timeout)
	{{- end}}
	if err != nil {
{{- if Prometheus}}
		clientCallsFor{{$serviceName}}.WithLabelValues(
			"{{.GetName}}", c.Encoding, "call_fail").Inc()
{{- end}}
		return // already logged
	}

	{{- if HasFullReply .}}
	resp = {{if HasPointerResultType .}}*{{end}}reply.GetResult()
	{{- end}}

{{- if Prometheus}}

	// report total time taken to Prometheus
	elapsed := time.Since(start).Seconds()
	clientRCTFor{{$serviceName}}.WithLabelValues("{{.GetName}}").Observe(elapsed)
	clientCallsFor{{$serviceName}}.WithLabelValues(
		"{{.GetName}}", c.Encoding, "success").Inc()
{{- end}}

	return
}
{{end -}}
{{- end -}}

{{- if Prometheus}}
func init() {
{{- range .Service}}
	// register metrics for service {{.GetName}}
	prometheus.MustRegister(clientRCTFor{{.GetName}})
	prometheus.MustRegister(serverHETFor{{.GetName}})
	prometheus.MustRegister(clientCallsFor{{.GetName}})
	prometheus.MustRegister(serverRequestsFor{{.GetName}})
{{- end}}
}
{{- end}}`
