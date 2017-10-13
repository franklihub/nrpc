// This code was autogenerated from helloworld.proto, do not edit.
package helloworld

import (
	"context"
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	nats "github.com/nats-io/go-nats"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rapidloop/nrpc"
)

// GreeterServer is the interface that providers of the service
// Greeter should implement.
type GreeterServer interface {
	SayHello(ctx context.Context, req HelloRequest) (resp HelloReply, err error)
}

var (
	// The request completion time, measured at client-side.
	clientRCTForGreeter = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "nrpc_client_request_completion_time_seconds",
			Help:       "The request completion time for Greeter calls, measured client-side.",
			Objectives: map[float64]float64{0.9: 0.01, 0.95: 0.01, 0.99: 0.001},
			ConstLabels: map[string]string{
				"service": "Greeter",
			},
		},
		[]string{"method"})

	// The handler execution time, measured at server-side.
	serverHETForGreeter = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "nrpc_server_handler_execution_time_seconds",
			Help:       "The handler execution time for Greeter calls, measured server-side.",
			Objectives: map[float64]float64{0.9: 0.01, 0.95: 0.01, 0.99: 0.001},
			ConstLabels: map[string]string{
				"service": "Greeter",
			},
		},
		[]string{"method"})

	// The counts of calls made by the client, classified by result type.
	clientCallsForGreeter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nrpc_client_calls_count",
			Help: "The count of calls made by the client.",
			ConstLabels: map[string]string{
				"service": "Greeter",
			},
		},
		[]string{"method", "encoding", "result_type"})

	// The counts of requests handled by the server, classified by result type.
	serverRequestsForGreeter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nrpc_server_requests_count",
			Help: "The count of requests handled by the server.",
			ConstLabels: map[string]string{
				"service": "Greeter",
			},
		},
		[]string{"method", "encoding", "result_type"})
)

// GreeterHandler provides a NATS subscription handler that can serve a
// subscription using a given GreeterServer implementation.
type GreeterHandler struct {
	ctx    context.Context
	nc     *nats.Conn
	server GreeterServer
}

func NewGreeterHandler(ctx context.Context, nc *nats.Conn, s GreeterServer) *GreeterHandler {
	return &GreeterHandler{
		ctx:    ctx,
		nc:     nc,
		server: s,
	}
}

func (h *GreeterHandler) Subject() string {
	return "Greeter.>"
}

func (h *GreeterHandler) Handler(msg *nats.Msg) {
	// extract method name & encoding from subject
	name, encoding, err := nrpc.ExtractFunctionNameAndEncoding(msg.Subject)

	// call handler and form response
	var resp proto.Message
	var replyError *nrpc.Error
	var elapsed float64
	switch name {
	case "SayHello":
		var req HelloRequest
		if err := nrpc.Unmarshal(encoding, msg.Data, &req); err != nil {
			log.Printf("SayHelloHandler: SayHello request unmarshal failed: %v", err)
			replyError = &nrpc.Error{
				Type: nrpc.Error_CLIENT,
				Message: "bad request received: " + err.Error(),
			}
			serverRequestsForGreeter.WithLabelValues(
				"SayHello", encoding, "unmarshal_fail").Inc()
		} else {
			start := time.Now()
			innerResp, err := h.server.SayHello(h.ctx, req)
			elapsed = time.Since(start).Seconds()
			if err != nil {
				log.Printf("SayHelloHandler: SayHello handler failed: %v", err)
				if e, ok := err.(*nrpc.Error); ok {
					replyError = e
				} else {
					replyError = &nrpc.Error{
						Type: nrpc.Error_CLIENT,
						Message: err.Error(),
					}
				}
				serverRequestsForGreeter.WithLabelValues(
					"SayHello", encoding, "handler_fail").Inc()
			} else {
				resp = &innerResp
			}
		}
	default:
		log.Printf("GreeterHandler: unknown name %q", name)
		replyError = &nrpc.Error{
			Type: nrpc.Error_CLIENT,
			Message: "unknown name: " + name,
		}
		serverRequestsForGreeter.WithLabelValues(
			"Greeter", encoding, "name_fail").Inc()
	}

	// encode and send response
	err = nrpc.Publish(resp, replyError, h.nc, msg.Reply, encoding) // error is logged
	if err != nil {
		serverRequestsForGreeter.WithLabelValues(
			name, encoding, "sendreply_fail").Inc()
	} else if replyError == nil {
		serverRequestsForGreeter.WithLabelValues(
			name, encoding, "success").Inc()
	}

	// report metric to Prometheus
	serverHETForGreeter.WithLabelValues(name).Observe(elapsed)
}

type GreeterClient struct {
	nc      *nats.Conn
	Subject string
	Encoding string
	Timeout time.Duration
}

func NewGreeterClient(nc *nats.Conn) *GreeterClient {
	return &GreeterClient{
		nc:      nc,
		Subject: "Greeter",
		Encoding: "protobuf",
		Timeout: 5 * time.Second,
	}
}

func (c *GreeterClient) SayHello(req HelloRequest) (resp HelloReply, err error) {
	start := time.Now()

	// call
	err = nrpc.Call(&req, &resp, c.nc, c.Subject+".SayHello", c.Encoding, c.Timeout)
	if err != nil {
		clientCallsForGreeter.WithLabelValues(
			"SayHello", c.Encoding, "call_fail").Inc()
		return // already logged
	}

	// report total time taken to Prometheus
	elapsed := time.Since(start).Seconds()
	clientRCTForGreeter.WithLabelValues("SayHello").Observe(elapsed)
	clientCallsForGreeter.WithLabelValues(
		"SayHello", c.Encoding, "success").Inc()

	return
}

func init() {
	// register metrics for service Greeter
	prometheus.MustRegister(clientRCTForGreeter)
	prometheus.MustRegister(serverHETForGreeter)
	prometheus.MustRegister(clientCallsForGreeter)
	prometheus.MustRegister(serverRequestsForGreeter)
}