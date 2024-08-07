// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package googleclientauthextension // import "github.com/GoogleCloudPlatform/opentelemetry-operations-go/extension/googleclientauthextension"

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/idtoken"
)

func TestPerRPCCredentials(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "testdata/fake_creds.json")
	ca := clientAuthenticator{config: &Config{
		Project:      "my-project",
		QuotaProject: "other-project",
		TokenType:    accessToken,
	}}
	err := ca.Start(context.Background(), nil)
	assert.NoError(t, err)

	perrpc, err := ca.PerRPCCredentials()
	assert.NotNil(t, perrpc)
	assert.NoError(t, err)
}

func TestPerRPCCredentialsWithIDToken(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "testdata/fake_isa_creds.json")
	ca := clientAuthenticator{
		config: &Config{
			Project:   "my-project",
			TokenType: idToken,
			Audience:  "http://example.com",
		},
		newIDTokenSource: idtoken.NewTokenSource,
	}
	err := ca.Start(context.Background(), nil)
	assert.NoError(t, err)

	perrpc, err := ca.PerRPCCredentials()
	assert.NotNil(t, perrpc)
	assert.NoError(t, err)
}

func TestPerRPCCredentialsNotStarted(t *testing.T) {
	ca := clientAuthenticator{config: &Config{
		Project:      "my-project",
		QuotaProject: "other-project",
		TokenType:    accessToken,
	}}
	perrpc, err := ca.PerRPCCredentials()
	assert.Nil(t, perrpc)
	assert.Error(t, err)
}
