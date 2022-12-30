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

	infra3Server1IPAddress := terraform.Output(t, terraformOptions, "infra3_server1_ip")
	infra3Server2IPAddress := terraform.Output(t, terraformOptions, "infra3_server2_ip")
	infra3MysqlEndpoint := terraform.Output(t, terraformOptions, "infra3_mysql_endpoint")
	infra3MysqlPassword := terraform.Output(t, terraformOptions, "infra3_mysql_password")
	infra3RancherURL := terraform.Output(t, terraformOptions, "infra3_rancher_url")

	noneOneIPAddressValidationResult := tools.CheckIPAddress(infra1Server1IPAddress)
	nodeTwoIPAddressValidationResult := tools.CheckIPAddress(infra1Server2IPAddress)

	assert.Equal(t, "valid", noneOneIPAddressValidationResult)
	assert.Equal(t, "valid", nodeTwoIPAddressValidationResult)

	actualHostNodeCount, _ := tools.SetupK3S(infra1MysqlPassword, infra1MysqlEndpoint, infra1RancherURL, infra1Server1IPAddress, infra1Server2IPAddress, viper.GetString("rancher_reproduction.email"), viper.GetString("rancher_reproduction.bootstrap_password"), viper.GetString("rancher_reproduction.version"), viper.GetString("rancher_reproduction.image_tag"), viper.GetString("rancher_reproduction.k3s_version"), "ha1-repro")
	actualTenantNodeCount, _ := tools.SetupK3S(infra2MysqlPassword, infra2MysqlEndpoint, infra2RancherURL, infra2Server1IPAddress, infra2Server2IPAddress, viper.GetString("rancher_head.email"), viper.GetString("rancher_head.bootstrap_password"), viper.GetString("rancher_head.version"), viper.GetString("rancher_head.image_tag"), viper.GetString("rancher_head.k3s_version"), "ha2-valid")
	actualExtraNodeCount, _ := tools.SetupK3S(infra3MysqlPassword, infra3MysqlEndpoint, infra3RancherURL, infra3Server1IPAddress, infra3Server2IPAddress, viper.GetString("rancher_extra.email"), viper.GetString("rancher_extra.bootstrap_password"), viper.GetString("rancher_extra.version"), viper.GetString("rancher_extra.image_tag"), viper.GetString("rancher_extra.k3s_version"), "ha3-extra")

	expectedNodeCount := 2

	assert.Equal(t, expectedNodeCount, actualHostNodeCount)
	assert.Equal(t, expectedNodeCount, actualTenantNodeCount)
	assert.Equal(t, expectedNodeCount, actualExtraNodeCount)

	t.Run("install ha1-repro rancher", TestInstallHA1)
	t.Run("install ha2-valid rancher", TestInstallHA2)
	t.Run("install ha3-extra rancher", TestInstallHA3)

	log.Printf("Rancher ha1-repro url repro: https://%s", infra1RancherURL)
	log.Printf("Rancher ha2-valid url valid: https://%s", infra2RancherURL)
	log.Printf("Rancher ha3-extra url valid: https://%s", infra3RancherURL)
}

func TestInstallHA1(t *testing.T) {

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{

		TerraformDir: "../modules/helm/ha1-repro",
		NoColor:      true,
	})

	terraform.InitAndApply(t, terraformOptions)
}

func TestInstallHA2(t *testing.T) {
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{

		TerraformDir: "../modules/helm/ha2-valid",
		NoColor:      true,
	})
	terraform.InitAndApply(t, terraformOptions)
}

func TestInstallHA3(t *testing.T) {
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{

		TerraformDir: "../modules/helm/ha3-extra",
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
	tools.RemoveFile("../../ha1-repro.yml")
	tools.RemoveFile("../../ha2-valid.yml")
	tools.RemoveFile("../../ha3-extra.yml")

	// Helm Host cleanup
	tools.RemoveFolder("../modules/helm/ha1-repro/.terraform")
	tools.RemoveFile("../modules/helm/ha1-repro/.terraform.lock.hcl")
	tools.RemoveFile("../modules/helm/ha1-repro/terraform.tfstate")
	tools.RemoveFile("../modules/helm/ha1-repro/terraform.tfstate.backup")
	tools.RemoveFile("../modules/helm/ha1-repro/terraform.tfvars")

	// Helm Tenant Cleanup
	tools.RemoveFolder("../modules/helm/ha2-valid/.terraform")
	tools.RemoveFile("../modules/helm/ha2-valid/.terraform.lock.hcl")
	tools.RemoveFile("../modules/helm/ha2-valid/terraform.tfstate")
	tools.RemoveFile("../modules/helm/ha2-valid/terraform.tfstate.backup")
	tools.RemoveFile("../modules/helm/ha2-valid/terraform.tfvars")

	// Helm Extra Cleanup
	tools.RemoveFolder("../modules/helm/ha3-extra/.terraform")
	tools.RemoveFile("../modules/helm/ha3-extra/.terraform.lock.hcl")
	tools.RemoveFile("../modules/helm/ha3-extra/terraform.tfstate")
	tools.RemoveFile("../modules/helm/ha3-extra/terraform.tfstate.backup")
	tools.RemoveFile("../modules/helm/ha3-extra/terraform.tfvars")

	// AWS Cleanup
	tools.RemoveFolder("../modules/aws/.terraform")
	tools.RemoveFile("../modules/aws/.terraform.lock.hcl")
	tools.RemoveFile("../modules/aws/terraform.tfstate")
	tools.RemoveFile("../modules/aws/terraform.tfstate.backup")
}
