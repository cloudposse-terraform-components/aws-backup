package test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/aws-component-helper"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponent(t *testing.T) {
	awsRegion := "us-east-2"

	fixture := helper.NewFixture(t, "../", awsRegion, "test/fixtures")

	defer fixture.TearDown()
	fixture.SetUp(&atmos.Options{})

	fixture.Suite("default", func(t *testing.T, suite *helper.Suite) {
		suite.Test(t, "basic", func(t *testing.T, atm *helper.Atmos) {
			planNameSuffix := strings.ToLower(random.UniqueId())
			inputs := map[string]interface{}{
				"plan_name_suffix": planNameSuffix,
			}

			defer atm.GetAndDestroy("aws-backup/basic", "default-test", inputs)
			component := atm.GetAndDeploy("aws-backup/basic", "default-test", inputs)
			assert.NotNil(t, component)

			vaultId := atm.Output(component, "backup_vault_id")
			assert.NotEmpty(t, vaultId)

			vaultArn := atm.Output(component, "backup_vault_arn")
			assert.Equal(t, fmt.Sprintf("arn:aws:backup:%s:%s:backup-vault:%s", awsRegion, fixture.AwsAccountId, vaultId), vaultArn)

			planArn := atm.Output(component, "backup_plan_arn")
			assert.NotEmpty(t, planArn)

			planVersion := atm.Output(component, "backup_plan_version")
			assert.NotEmpty(t, planVersion)

			planSelectionId := atm.Output(component, "backup_selection_id")
			assert.NotEmpty(t, planSelectionId)

			client := NewBackupClient(t, awsRegion)
			vault, err := client.DescribeBackupVault(context.Background(), &backup.DescribeBackupVaultInput{
				BackupVaultName: &vaultId,
			})
			require.NoError(t, err)

			assert.Equal(t, vaultId, *vault.BackupVaultName)
			assert.Equal(t, vaultArn, *vault.BackupVaultArn)
			assert.EqualValues(t, "BACKUP_VAULT", vault.VaultType)
			assert.EqualValues(t, "", vault.VaultState)
			assert.False(t, *vault.Locked)
		})
	})
}

func NewBackupClient(t *testing.T, region string) *backup.Client {
	client, err := NewBackupClientE(t, region)
	require.NoError(t, err)

	return client
}

func NewBackupClientE(t *testing.T, region string) (*backup.Client, error) {
	sess, err := aws.NewAuthenticatedSession(region)
	if err != nil {
		return nil, err
	}
	return backup.NewFromConfig(*sess), nil
}
