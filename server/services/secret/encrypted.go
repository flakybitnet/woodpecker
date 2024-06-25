// Copyright 2023 Woodpecker Authors
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
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.woodpecker-ci.org/woodpecker/v2/server/model"
	"go.woodpecker-ci.org/woodpecker/v2/server/services/encryption"
)

const (
	secretValueTemplate = "%s_"
)

type encryptedSecretService struct {
	secretSvc     Service
	encryptionSvc encryption.Service
	locked        bool
}

func NewEncrypted(secretService Service, encryptionService encryption.Service) *encryptedSecretService {
	return &encryptedSecretService{
		secretSvc:     secretService,
		encryptionSvc: encryptionService,
	}
}

func (ess *encryptedSecretService) isLocked() bool {
	return ess.locked
}

func (ess *encryptedSecretService) lock() {
	ess.locked = true
}

func (ess *encryptedSecretService) unlock() {
	ess.locked = false
}

func (ess *encryptedSecretService) encryptSecret(secret *model.Secret) error {
	if ess.isEncoded(secret.Value) {
		return nil
	}
	log.Debug().Int64("id", secret.ID).Str("name", secret.Name).Msg("encryption")

	encryptedValue, err := ess.encryptionSvc.Encrypt(secret.Value, secret.Name)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret id=%d: %w", secret.ID, err)
	}
	encodedValue := ess.encodeSecretValue(encryptedValue)

	secret.Value = encodedValue
	return nil
}

func (ess *encryptedSecretService) encodeSecretValue(value string) string {
	return ess.header() + value
}

func (ess *encryptedSecretService) decryptList(secrets []*model.Secret) error {
	for _, secret := range secrets {
		err := ess.decryptSecret(secret)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ess *encryptedSecretService) decryptSecret(secret *model.Secret) error {
	if !ess.isEncoded(secret.Value) {
		return nil
	}
	log.Debug().Int64("id", secret.ID).Str("name", secret.Name).Msg("decryption")

	decodedValue := ess.decodeSecretValue(secret.Value)
	decryptedValue, err := ess.encryptionSvc.Decrypt(decodedValue, secret.Name)
	if err != nil {
		return fmt.Errorf("failed to decrypt secret id=%d: %w", secret.ID, err)
	}

	secret.Value = decryptedValue
	return nil
}

func (ess *encryptedSecretService) decodeSecretValue(value string) string {
	return strings.TrimPrefix(value, ess.header())
}

func (ess *encryptedSecretService) isEncoded(value string) bool {
	return strings.HasPrefix(value, ess.header())
}

func (ess *encryptedSecretService) header() string {
	return fmt.Sprintf(secretValueTemplate, ess.encryptionSvc.Algo())
}

// Service (server/services/secret/service.go) interface implementation

func (ess *encryptedSecretService) SecretFind(repo *model.Repo, name string) (*model.Secret, error) {
	if ess.isLocked() {
		return nil, newServiceLockedError()
	}

	secret, err := ess.secretSvc.SecretFind(repo, name)
	if err != nil {
		return nil, err
	}
	err = ess.decryptSecret(secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (ess *encryptedSecretService) SecretList(repo *model.Repo, listOpt *model.ListOptions) ([]*model.Secret, error) {
	if ess.isLocked() {
		return nil, newServiceLockedError()
	}

	secrets, err := ess.secretSvc.SecretList(repo, listOpt)
	if err != nil {
		return nil, err
	}
	err = ess.decryptList(secrets)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

func (ess *encryptedSecretService) SecretListPipeline(repo *model.Repo, pipeline *model.Pipeline, listOpt *model.ListOptions) ([]*model.Secret, error) {
	if ess.isLocked() {
		return nil, newServiceLockedError()
	}

	secrets, err := ess.secretSvc.SecretListPipeline(repo, pipeline, listOpt)
	if err != nil {
		return nil, err
	}
	err = ess.decryptList(secrets)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

func (ess *encryptedSecretService) SecretCreate(repo *model.Repo, in *model.Secret) error {
	if ess.isLocked() {
		return newServiceLockedError()
	}

	err := ess.encryptSecret(in)
	if err != nil {
		return err
	}
	err = ess.secretSvc.SecretCreate(repo, in)
	if err != nil {
		return err
	}
	return nil
}

func (ess *encryptedSecretService) SecretUpdate(repo *model.Repo, in *model.Secret) error {
	if ess.isLocked() {
		return newServiceLockedError()
	}

	err := ess.encryptSecret(in)
	if err != nil {
		return err
	}
	err = ess.secretSvc.SecretUpdate(repo, in)
	if err != nil {
		return err
	}
	return nil
}

func (ess *encryptedSecretService) SecretDelete(repo *model.Repo, name string) error {
	if ess.isLocked() {
		return newServiceLockedError()
	}

	return ess.secretSvc.SecretDelete(repo, name)
}

func (ess *encryptedSecretService) OrgSecretFind(owner int64, name string) (*model.Secret, error) {
	if ess.isLocked() {
		return nil, newServiceLockedError()
	}

	secret, err := ess.secretSvc.OrgSecretFind(owner, name)
	if err != nil {
		return nil, err
	}
	err = ess.decryptSecret(secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (ess *encryptedSecretService) OrgSecretList(owner int64, listOpt *model.ListOptions) ([]*model.Secret, error) {
	if ess.isLocked() {
		return nil, newServiceLockedError()
	}

	secrets, err := ess.secretSvc.OrgSecretList(owner, listOpt)
	if err != nil {
		return nil, err
	}
	err = ess.decryptList(secrets)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

func (ess *encryptedSecretService) OrgSecretCreate(owner int64, in *model.Secret) error {
	if ess.isLocked() {
		return newServiceLockedError()
	}

	err := ess.encryptSecret(in)
	if err != nil {
		return err
	}
	err = ess.secretSvc.OrgSecretCreate(owner, in)
	if err != nil {
		return err
	}
	return nil
}

func (ess *encryptedSecretService) OrgSecretUpdate(owner int64, in *model.Secret) error {
	if ess.isLocked() {
		return newServiceLockedError()
	}

	err := ess.encryptSecret(in)
	if err != nil {
		return err
	}
	err = ess.secretSvc.OrgSecretUpdate(owner, in)
	if err != nil {
		return err
	}
	return nil
}

func (ess *encryptedSecretService) OrgSecretDelete(owner int64, name string) error {
	if ess.isLocked() {
		return newServiceLockedError()
	}

	return ess.secretSvc.OrgSecretDelete(owner, name)
}

func (ess *encryptedSecretService) GlobalSecretFind(owner string) (*model.Secret, error) {
	if ess.isLocked() {
		return nil, newServiceLockedError()
	}

	secret, err := ess.secretSvc.GlobalSecretFind(owner)
	if err != nil {
		return nil, err
	}
	err = ess.decryptSecret(secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (ess *encryptedSecretService) GlobalSecretList(listOpt *model.ListOptions) ([]*model.Secret, error) {
	if ess.isLocked() {
		return nil, newServiceLockedError()
	}

	secrets, err := ess.secretSvc.GlobalSecretList(listOpt)
	if err != nil {
		return nil, err
	}
	err = ess.decryptList(secrets)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

func (ess *encryptedSecretService) GlobalSecretCreate(in *model.Secret) error {
	if ess.isLocked() {
		return newServiceLockedError()
	}

	err := ess.encryptSecret(in)
	if err != nil {
		return err
	}
	err = ess.secretSvc.GlobalSecretCreate(in)
	if err != nil {
		return err
	}
	return nil
}

func (ess *encryptedSecretService) GlobalSecretUpdate(in *model.Secret) error {
	if ess.isLocked() {
		return newServiceLockedError()
	}

	err := ess.encryptSecret(in)
	if err != nil {
		return err
	}
	err = ess.secretSvc.GlobalSecretUpdate(in)
	if err != nil {
		return err
	}
	return nil
}

func (ess *encryptedSecretService) GlobalSecretDelete(name string) error {
	if ess.isLocked() {
		return newServiceLockedError()
	}

	return ess.secretSvc.GlobalSecretDelete(name)
}

func newServiceLockedError() error {
	return fmt.Errorf("service is locked")
}

type EncryptedSecretMigrationAgent struct {
	ess   *encryptedSecretService
	store model.SecretStore
}

func NewMigration(ess *encryptedSecretService, store model.SecretStore) *EncryptedSecretMigrationAgent {
	ess.lock()

	return &EncryptedSecretMigrationAgent{
		ess:   ess,
		store: store,
	}
}

func (esma *EncryptedSecretMigrationAgent) EncryptAll() error {
	log.Info().Msg("encrypting all secrets")

	secrets, err := esma.store.SecretListAll()
	if err != nil {
		return newAllEncryptionError(err)
	}

	for _, secret := range secrets {
		if err := esma.ess.encryptSecret(secret); err != nil {
			return newAllEncryptionError(err)
		}
		if err := esma.store.SecretUpdate(secret); err != nil {
			return newAllEncryptionError(err)
		}
	}

	log.Info().Msg("all secrets are encrypted")
	esma.ess.unlock()
	return nil
}

// DecryptAll call from CLI
func (esma *EncryptedSecretMigrationAgent) DecryptAll() error {
	log.Info().Msg("decrypting all secrets")

	secrets, err := esma.store.SecretListAll()
	if err != nil {
		return newAllDecryptionError(err)
	}

	for _, secret := range secrets {
		if err := esma.ess.decryptSecret(secret); err != nil {
			return newAllDecryptionError(err)
		}
		if err := esma.store.SecretUpdate(secret); err != nil {
			return newAllDecryptionError(err)
		}
	}

	log.Info().Msg("all secrets are decrypted, remove the encryption key and restart")
	return nil
}

func newAllEncryptionError(e error) error {
	return fmt.Errorf("cannot encrypt secrets: %w", e)
}

func newAllDecryptionError(e error) error {
	return fmt.Errorf("cannot decrypt secrets: %w", e)
}
