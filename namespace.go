package eventhub

//	MIT License
//
//	Copyright (c) Microsoft Corporation. All rights reserved.
//
//	Permission is hereby granted, free of charge, to any person obtaining a copy
//	of this software and associated documentation files (the "Software"), to deal
//	in the Software without restriction, including without limitation the rights
//	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//	copies of the Software, and to permit persons to whom the Software is
//	furnished to do so, subject to the following conditions:
//
//	The above copyright notice and this permission notice shall be included in all
//	copies or substantial portions of the Software.
//
//	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//	SOFTWARE

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/Azure/azure-amqp-common-go/v3/auth"
	"github.com/Azure/azure-amqp-common-go/v3/cbs"
	"github.com/Azure/azure-amqp-common-go/v3/conn"
	"github.com/Azure/azure-amqp-common-go/v3/sas"
	"github.com/Azure/go-amqp"
	"github.com/Azure/go-autorest/autorest/azure"
	websocket2 "nhooyr.io/websocket"
)

type (
	namespace struct {
		name          string
		tokenProvider auth.TokenProvider
		host          string
		useWebSocket  bool
		proxy 		*url.URL
	}

	// namespaceOption provides structure for configuring a new Event Hub namespace
	namespaceOption func(h *namespace) error
)

// newNamespaceWithConnectionString configures a namespace with the information provided in a Service Bus connection string
func namespaceWithConnectionString(connStr string) namespaceOption {
	return func(ns *namespace) error {
		parsed, err := conn.ParsedConnectionFromStr(connStr)
		if err != nil {
			return err
		}
		ns.name = parsed.Namespace
		ns.host = parsed.Host
		provider, err := sas.NewTokenProvider(sas.TokenProviderWithKey(parsed.KeyName, parsed.Key))
		if err != nil {
			return err
		}
		ns.tokenProvider = provider
		return nil
	}
}

func namespaceWithAzureEnvironment(name string, tokenProvider auth.TokenProvider, env azure.Environment) namespaceOption {
	return func(ns *namespace) error {
		ns.name = name
		ns.tokenProvider = tokenProvider
		ns.host = "amqps://" + ns.name + "." + env.ServiceBusEndpointSuffix
		return nil
	}
}

// newNamespace creates a new namespace configured through NamespaceOption(s)
func newNamespace(opts ...namespaceOption) (*namespace, error) {
	ns := &namespace{}

	for _, opt := range opts {
		err := opt(ns)
		if err != nil {
			return nil, err
		}
	}

	return ns, nil
}

func (ns *namespace) newConnection() (*amqp.Client, error) {
	host := ns.getAmqpsHostURI()

	defaultConnOptions := []amqp.ConnOption{
		amqp.ConnSASLAnonymous(),
		amqp.ConnProperty("product", "MSGolangClient"),
		amqp.ConnProperty("version", Version),
		amqp.ConnProperty("platform", runtime.GOOS),
		amqp.ConnProperty("framework", runtime.Version()),
		amqp.ConnProperty("user-agent", rootUserAgent),
	}

	if ns.useWebSocket {
		trimmedHost := strings.TrimPrefix(ns.host, "amqps://")
		header := http.Header{}
		header.Add("origin","http://localhost/")

		if ns.proxy != nil {

			dialer := &websocket2.DialOptions{
				HTTPClient: &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(ns.proxy)}},
				HTTPHeader:           header,
				Subprotocols:         []string{"amqp"},
			}
			fmt.Println("socket --", ns.proxy)
			ctx := context.Background()
			conn , _ , err := websocket2.Dial(ctx,"wss://"+trimmedHost+"/$servicebus/websocket",dialer)
			if err != nil {
				return nil, err
			}
			//reader, _ , err := conn.Reader(ctx)
			//if err != nil {
			//	return nil, err
			//}

			netConn :=  websocket2.NetConn(ctx,conn,2)

			return amqp.New(netConn, append(defaultConnOptions, amqp.ConnServerHostname(trimmedHost))...)
		}else {
			dialer := &websocket2.DialOptions{
				HTTPHeader:           header,
				Subprotocols:         []string{"amqp"},
			}
			fmt.Println("socket2 --")
			ctx := context.Background()
			conn , _ , err := websocket2.Dial(ctx,"wss://"+trimmedHost+"/$servicebus/websocket",dialer)
			if err != nil {
				return nil, err
			}
			//reader, _ , err := conn.Reader(ctx)
			//if err != nil {
			//	return nil, err
			//}

			netConn :=  websocket2.NetConn(ctx,conn,2)

			return amqp.New(netConn, append(defaultConnOptions, amqp.ConnServerHostname(trimmedHost))...)
		}
	}

	return amqp.Dial(host, defaultConnOptions...)
}

func (ns *namespace) negotiateClaim(ctx context.Context, conn *amqp.Client, entityPath string) error {
	span, ctx := ns.startSpanFromContext(ctx, "eh.namespace.negotiateClaim")
	defer span.End()

	audience := ns.getEntityAudience(entityPath)
	return cbs.NegotiateClaim(ctx, audience, conn, ns.tokenProvider)
}

func (ns *namespace) getAmqpsHostURI() string {
	return ns.host + "/"
}

func (ns *namespace) getAmqpHostURI() string {
	return strings.Replace(ns.getAmqpsHostURI(), "amqps", "amqp", 1)
}

func (ns *namespace) getEntityAudience(entityPath string) string {
	return ns.getAmqpsHostURI() + entityPath
}

func (ns *namespace) getHTTPSHostURI() string {
	return strings.Replace(ns.getAmqpsHostURI(), "amqps", "https", 1)
}
