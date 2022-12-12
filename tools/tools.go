package toolkit

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/spf13/viper"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyz"

type Tools struct{}

func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}
	return string(s)
}

func (t *Tools) SetupK3S(mysqlPassword string, mysqlEndpoint string, rancherURL string, node1IP string, node2IP string, rancherEmail string, rancherBsPw string, rancherVersion string, rancherImage string, k3sVersion string, rancherType string) (int, string) {

	nodeOneCommand := fmt.Sprintf(`curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION='%s' sh -s - server --token=SECRET --datastore-endpoint='mysql://tfadmin:%s@tcp(%s)/k3s' --tls-san %s --node-external-ip %s`, k3sVersion, mysqlPassword, mysqlEndpoint, rancherURL, node1IP)

	var _ = t.RunCommand(nodeOneCommand, node1IP)

	token := t.RunCommand("sudo cat /var/lib/rancher/k3s/server/token", node1IP)
	serverKubeConfig := t.RunCommand("sudo cat /etc/rancher/k3s/k3s.yaml", node1IP)

	time.Sleep(10 * time.Second)

	nodeTwoCommand := fmt.Sprintf(`curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION='%s' sh -s - server --token=%s --datastore-endpoint='mysql://tfadmin:%s@tcp(%s)/k3s' --tls-san %s --node-external-ip %s`, k3sVersion, token, mysqlPassword, mysqlEndpoint, rancherURL, node2IP)
	var _ = t.RunCommand(nodeTwoCommand, node2IP)

	time.Sleep(10 * time.Second)

	wcResponse := t.RunCommand("sudo k3s kubectl get nodes | wc -l", node1IP)
	actualNodeCount, err := strconv.Atoi(wcResponse)
	actualNodeCount = actualNodeCount - 1

	if err != nil {
		log.Println(err)
	}

	kubeConf := []byte(serverKubeConfig)

	configIP := fmt.Sprintf("https://%s:6443", node1IP)
	output := bytes.Replace(kubeConf, []byte("https://127.0.0.1:6443"), []byte(configIP), -1)

	if rancherType == "ha1" {
		err = os.WriteFile("../../ha1.yml", output, 0644)
		if err != nil {
			log.Println("failed creating ha1 config:", err)
		}
	} else if rancherType == "ha2" {
		err = os.WriteFile("../../ha2.yml", output, 0644)
		if err != nil {
			log.Println("failed creating ha2 config:", err)
		}
	} else {
		log.Fatal("expecting either ha1 or ha2 for rancher type")
	}

	tfvarFile := fmt.Sprintf("rancher_url = \"%s\"\nbootstrap_password = \"%s\"\nemail = \"%s\"\nrancher_version = \"%s\"\nimage_tag = \"%s\"", rancherURL, rancherBsPw, rancherEmail, rancherVersion, rancherImage)
	tfvarFileBytes := []byte(tfvarFile)

	if rancherType == "ha1" {
		err = os.WriteFile("../modules/helm/ha1/terraform.tfvars", tfvarFileBytes, 0644)

		if err != nil {
			log.Println("failed creating ha1 tfvars:", err)
		}
	} else if rancherType == "ha2" {
		err = os.WriteFile("../modules/helm/ha2/terraform.tfvars", tfvarFileBytes, 0644)

		if err != nil {
			log.Println("failed creating ha2 tfvars:", err)
		}
	} else {
		log.Fatal("expecting either ha1 or ha2 for rancher type")
	}

	return actualNodeCount, configIP
}

func (t *Tools) RunCommand(cmd string, pubIP string) string {

	path := viper.GetString("local.pem_path")

	dialIP := fmt.Sprintf("%s:22", pubIP)

	pemBytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		log.Fatalf("parse key failed:%v", err)
	}
	config := &ssh.ClientConfig{
		User:            "ubuntu",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", dialIP, config)
	if err != nil {
		log.Fatalf("dial failed:%v", err)
	}
	defer func(conn *ssh.Client) {
		err := conn.Close()
		if err != nil {
			log.Println(err)
		}
	}(conn)
	session, err := conn.NewSession()
	if err != nil {
		log.Fatalf("session failed:%v", err)
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil {
			log.Println(err)
		}
	}(session)
	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	err = session.Run(cmd)
	if err != nil {
		log.Fatalf("Run failed:%v", err)
	}

	stringOut := stdoutBuf.String()

	stringOut = strings.TrimRight(stringOut, "\r\n")

	return stringOut
}

func (t *Tools) CheckIPAddress(ip string) string {
	if net.ParseIP(ip) == nil {
		return "invalid"
	} else {
		return "valid"
	}
}

func (t *Tools) RemoveFile(filePath string) {
	err := os.Remove(filePath)
	if err != nil {
		log.Println(err)
	}
}

func (t *Tools) RemoveFolder(folderPath string) {
	err := os.RemoveAll(folderPath)
	if err != nil {
		log.Println(err)
	}
}
