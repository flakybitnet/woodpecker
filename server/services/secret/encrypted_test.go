// Copyright 2024 Woodpecker Authors
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

package secret

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"go.woodpecker-ci.org/woodpecker/v2/server/services/encryption"
	"go.woodpecker-ci.org/woodpecker/v2/server/services/secret/mocks"
	mocks_store "go.woodpecker-ci.org/woodpecker/v2/server/store/mocks"
	"strings"
	"testing"
)

type testEncSvc struct {
}

func newTestEncSvc() encryption.Service {
	return &testEncSvc{}
}

func (e *testEncSvc) Algo() string {
	return "enc"
}

func (e *testEncSvc) Encrypt(plaintext, associatedData string) (string, error) {
	return strings.Join([]string{"encrypted", plaintext, "encrypted"}, "-"), nil
}

func (e *testEncSvc) Decrypt(ciphertext, associatedData string) (string, error) {
	s := strings.TrimPrefix(ciphertext, "encrypted-")
	return strings.TrimSuffix(s, "-encrypted"), nil
}

func TestSecretFind(t *testing.T) {
	secretSvc := mocks.NewService(t)
	secretSvc.On("SecretFind", mock.Anything, mock.Anything).Once().
		Return(&model.Secret{Value: "enc_encrypted-awsaccesskeyexample-encrypted"}, nil)
	ess := NewEncrypted(secretSvc, newTestEncSvc())

	secret, err := ess.SecretFind(nil, "sec")
	assert.NoError(t, err)
	assert.Equal(t, "awsaccesskeyexample", secret.Value)
}

func TestSecretCreate(t *testing.T) {
	secretSvc := mocks.NewService(t)
	secretSvc.On("SecretCreate", mock.Anything, mock.Anything).Once().Return(nil).
		Run(func(args mock.Arguments) {
			secret := args.Get(1).(*model.Secret)
			assert.NotNil(t, secret)
			assert.Equal(t, "enc_encrypted-awsaccesskeyexample-encrypted", secret.Value)
		})
	ess := NewEncrypted(secretSvc, newTestEncSvc())

	err := ess.SecretCreate(nil, &model.Secret{Value: "awsaccesskeyexample"})
	assert.NoError(t, err)
}

func TestMigrationEncrypt(t *testing.T) {
	store := mocks_store.NewStore(t)
	secretSvc := mocks.NewService(t)
	ess := NewEncrypted(secretSvc, newTestEncSvc())

	store.On("SecretListAll").Once().
		Return([]*model.Secret{{Value: "supersec"}}, nil)

	store.On("SecretUpdate", mock.Anything).Once().Return(nil).
		Run(func(args mock.Arguments) {
			secret := args.Get(0).(*model.Secret)
			assert.NotNil(t, secret)
			assert.Equal(t, "enc_encrypted-supersec-encrypted", secret.Value)
		})

	migrationAgent := NewMigration(ess, store)
	assert.True(t, ess.isLocked())

	err := migrationAgent.EncryptAll()
	assert.NoError(t, err)
	assert.False(t, ess.isLocked())
}

func TestMigrationDecrypt(t *testing.T) {
	store := mocks_store.NewStore(t)
	secretSvc := mocks.NewService(t)
	ess := NewEncrypted(secretSvc, newTestEncSvc())

	store.On("SecretListAll").Once().
		Return([]*model.Secret{{Value: "enc_encrypted-supersec-encrypted"}}, nil)

	store.On("SecretUpdate", mock.Anything).Once().Return(nil).
		Run(func(args mock.Arguments) {
			secret := args.Get(0).(*model.Secret)
			assert.NotNil(t, secret)
			assert.Equal(t, "supersec", secret.Value)
		})

	migrationAgent := NewMigration(ess, store)
	assert.True(t, ess.isLocked())

	err := migrationAgent.DecryptAll()
	assert.NoError(t, err)
	assert.True(t, ess.isLocked())
}
