package test

import (
	toolkit "github.com/brudnak/rancher-k3s-ha/tools"
	"github.com/spf13/viper"
	"log"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

func TestK3sHa(t *testing.T) {

	viper.AddConfigPath("../../")
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	err := viper.ReadInConfig()

	if err != nil {
		log.Println("error reading config:", err)
	}

	var tools toolkit.Tools

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{

		TerraformDir: "../modules/aws",
		NoColor:      true,
	})

	terraform.InitAndApply(t, terraformOptions)

	infra1Server1IPAddress := terraform.Output(t, terraformOptions, "infra1_server1_ip")
	infra1Server2IPAddress := terraform.Output(t, terraformOptions, "infra1_server2_ip")
	infra1MysqlEndpoint := terraform.Output(t, terraformOptions, "infra1_mysql_endpoint")
	infra1MysqlPassword := terraform.Output(t, terraformOptions, "infra1_mysql_password")
	infra1RancherURL := terraform.Output(t, terraformOptions, "infra1_rancher_url")

	infra2Server1IPAddress := terraform.Output(t, terraformOptions, "infra2_server1_ip")
	infra2Server2IPAddress := terraform.Output(t, terraformOptions, "infra2_server2_ip")
	infra2MysqlEndpoint := terraform.Output(t, terraformOptions, "infra2_mysql_endpoint")
	infra2MysqlPassword := terraform.Output(t, terraformOptions, "infra2_mysql_password")
	infra2RancherURL := terraform.Output(t, terraformOptions, "infra2_rancher_url")

	noneOneIPAddressValidationResult := tools.CheckIPAddress(infra1Server1IPAddress)
	nodeTwoIPAddressValidationResult := tools.CheckIPAddress(infra1Server2IPAddress)

	assert.Equal(t, "valid", noneOneIPAddressValidationResult)
	assert.Equal(t, "valid", nodeTwoIPAddressValidationResult)

	actualHostNodeCount, _ := tools.SetupK3S(infra1MysqlPassword, infra1MysqlEndpoint, infra1RancherURL, infra1Server1IPAddress, infra1Server2IPAddress, "ha1")
	actualTenantNodeCount, _ := tools.SetupK3S(infra2MysqlPassword, infra2MysqlEndpoint, infra2RancherURL, infra2Server1IPAddress, infra2Server2IPAddress, "ha2")

	expectedNodeCount := 2

	assert.Equal(t, expectedNodeCount, actualHostNodeCount)
	assert.Equal(t, expectedNodeCount, actualTenantNodeCount)

	t.Run("install ha1 rancher", TestInstallHA1)
	t.Run("install ha2 rancher", TestInstallHA2)

	log.Printf("Rancher ha1 url: https://%s", infra1RancherURL)
	log.Printf("Rancher ha2 url: https://%s", infra2RancherURL)
}

func TestInstallHA1(t *testing.T) {

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{

		TerraformDir: "../modules/helm/ha1",
		NoColor:      true,
	})

	terraform.InitAndApply(t, terraformOptions)
}

func TestInstallHA2(t *testing.T) {
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{

		TerraformDir: "../modules/helm/ha2",
		NoColor:      true,
	})
	terraform.InitAndApply(t, terraformOptions)
}

func TestHostCleanup(t *testing.T) {
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../modules/aws",
		NoColor:      true,
	})

	terraform.Destroy(t, terraformOptions)

	var tools toolkit.Tools

	// Kubeconfig files
	tools.RemoveFile("../../ha1.yml")
	tools.RemoveFile("../../ha2.yml")

	// Helm Host cleanup
	tools.RemoveFolder("../modules/helm/ha1/.terraform")
	tools.RemoveFile("../modules/helm/ha1/.terraform.lock.hcl")
	tools.RemoveFile("../modules/helm/ha1/terraform.tfstate")
	tools.RemoveFile("../modules/helm/ha1/terraform.tfstate.backup")
	tools.RemoveFile("../modules/helm/ha1/terraform.tfvars")

	// Helm Tenant Cleanup
	tools.RemoveFolder("../modules/helm/ha2/.terraform")
	tools.RemoveFile("../modules/helm/ha2/.terraform.lock.hcl")
	tools.RemoveFile("../modules/helm/ha2/terraform.tfstate")
	tools.RemoveFile("../modules/helm/ha2/terraform.tfstate.backup")
	tools.RemoveFile("../modules/helm/ha2/terraform.tfvars")

	// AWS Cleanup
	tools.RemoveFolder("../modules/aws/.terraform")
	tools.RemoveFile("../modules/aws/.terraform.lock.hcl")
	tools.RemoveFile("../modules/aws/terraform.tfstate")
	tools.RemoveFile("../modules/aws/terraform.tfstate.backup")
}
