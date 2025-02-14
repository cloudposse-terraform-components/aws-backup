package test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	awshelper "github.com/cloudposse/test-helpers/pkg/aws"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ComponentSuite struct {
	helper.TestSuite
}

func (s *ComponentSuite) TestBasic() {
	const component = "aws-backup/basic"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	planNameSuffix := strings.ToLower(random.UniqueId())
	inputs := map[string]interface{}{
		"plan_name_suffix": planNameSuffix,
	}

	defer s.DestroyAtmosComponent(s.T(), component, stack, &inputs)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, &inputs)
	assert.NotNil(s.T(), options)

	vaultId := atmos.Output(s.T(), options, "backup_vault_id")
	assert.NotEmpty(s.T(), vaultId)

	AwsAccountId := aws.GetAccountId(s.T())

	vaultArn := atmos.Output(s.T(), options, "backup_vault_arn")
	assert.Equal(s.T(), fmt.Sprintf("arn:aws:backup:%s:%s:backup-vault:%s", awsRegion, AwsAccountId, vaultId), vaultArn)

	planArn := atmos.Output(s.T(), options, "backup_plan_arn")
	assert.NotEmpty(s.T(), planArn)

	planVersion := atmos.Output(s.T(), options, "backup_plan_version")
	assert.NotEmpty(s.T(), planVersion)

	planSelectionId := atmos.Output(s.T(), options, "backup_selection_id")
	assert.NotEmpty(s.T(), planSelectionId)

	client := awshelper.NewBackupClient(s.T(), awsRegion)
	vault, err := client.DescribeBackupVault(context.Background(), &backup.DescribeBackupVaultInput{
		BackupVaultName: &vaultId,
	})
	require.NoError(s.T(), err)

	assert.Equal(s.T(), vaultId, *vault.BackupVaultName)
	assert.Equal(s.T(), vaultArn, *vault.BackupVaultArn)
	assert.EqualValues(s.T(), "BACKUP_VAULT", vault.VaultType)
	assert.EqualValues(s.T(), "", vault.VaultState)
	assert.False(s.T(), *vault.Locked)

	s.DriftTest(component, stack, &inputs)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "aws-backup/disabled"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	planNameSuffix := strings.ToLower(random.UniqueId())
	inputs := map[string]interface{}{
		"plan_name_suffix": planNameSuffix,
	}
	s.VerifyEnabledFlag(component, stack, &inputs)
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)
	helper.Run(t, suite)
}
