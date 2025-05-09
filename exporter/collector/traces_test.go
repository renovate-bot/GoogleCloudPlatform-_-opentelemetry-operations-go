// Copyright 2021 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"context"
	"net"
	"testing"
	"time"

	"cloud.google.com/go/trace/apiv2/tracepb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type testServer struct {
	reqCh chan *tracepb.BatchWriteSpansRequest
}

func (ts *testServer) BatchWriteSpans(ctx context.Context, req *tracepb.BatchWriteSpansRequest) (*emptypb.Empty, error) {
	go func() { ts.reqCh <- req }()
	return &emptypb.Empty{}, nil
}

// Creates a new span.
func (ts *testServer) CreateSpan(context.Context, *tracepb.Span) (*tracepb.Span, error) {
	return nil, nil
}

func TestGoogleCloudTraceExport(t *testing.T) {
	type testCase struct {
		name               string
		expectedErr        string
		expectedServiceKey string
		cfg                Config
	}

	testCases := []testCase{
		{
			name: "Standard",
			cfg: Config{
				ProjectID: "idk",
				TraceConfig: TraceConfig{
					ClientConfig: ClientConfig{
						Endpoint:    "127.0.0.1:8080",
						UseInsecure: true,
					},
				},
			},
			expectedServiceKey: "service.name",
		},
		{
			name: "With Custom Mapping",
			cfg: Config{
				ProjectID: "idk",
				TraceConfig: TraceConfig{
					ClientConfig: ClientConfig{
						Endpoint:    "127.0.0.1:8080",
						UseInsecure: true,
					},
					AttributeMappings: []AttributeMapping{
						{
							Key:         "service.name",
							Replacement: "g.co/gae/app/module",
						},
					},
				},
			},
			expectedServiceKey: "g.co/gae/app/module",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			srv := grpc.NewServer()
			reqCh := make(chan *tracepb.BatchWriteSpansRequest)
			tracepb.RegisterTraceServiceServer(srv, &testServer{reqCh: reqCh})

			lis, err := net.Listen("tcp", "localhost:8080")
			require.NoError(t, err)
			defer lis.Close()

			//nolint:errcheck
			go srv.Serve(lis)
			sde, err := NewGoogleCloudTracesExporter(ctx, test.cfg, newTestExporterSettings(), DefaultTimeout)
			require.NoError(t, err)
			err = sde.Start(ctx, componenttest.NewNopHost())
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
				return
			}
			require.NoError(t, err)
			defer func() { require.NoError(t, sde.Shutdown(ctx)) }()

			testTime := time.Now()
			spanName := "foobar"

			resource := pcommon.NewResource()
			traces := ptrace.NewTraces()
			rspans := traces.ResourceSpans().AppendEmpty()
			resource.CopyTo(rspans.Resource())
			ispans := rspans.ScopeSpans().AppendEmpty()
			span := ispans.Spans().AppendEmpty()
			span.SetName(spanName)
			span.SetStartTimestamp(pcommon.NewTimestampFromTime(testTime))
			span.Attributes().PutStr("service.name", "myservice")
			err = sde.PushTraces(ctx, traces)
			assert.NoError(t, err)

			r := <-reqCh
			assert.Len(t, r.Spans, 1)
			assert.Equal(t, spanName, r.Spans[0].GetDisplayName().Value)
			_, ok := r.Spans[0].GetAttributes().GetAttributeMap()[test.expectedServiceKey]
			assert.True(t, ok)
			assert.Equal(t, timestamppb.New(testTime), r.Spans[0].StartTime)
		})
	}
}
