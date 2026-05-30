package cmd

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/mailcli/test"
)

const mailRepo = "docker.io/mailserver/docker-mailserver"
const mailRepoTag = "15.1.0"
const mailHostname = "mail.test.local"
const smtpDockerPort = 31025
const imapDockerPort = 31143
const sslDockerPort = 31465
const tlsDockerPort = 31587
const imapsDockerPort = 31993
const containerWaitTimeout = 120

var mailContainerName string
var mailContainer *dockertest.Resource
var mailDockerServer = "127.0.0.1"

// prepareMailContainer starts a docker-mailserver container for integration tests
func prepareMailContainer() (container *dockertest.Resource, err error) {
	if os.Getenv("SKIP_MAIL") != "" {
		err = fmt.Errorf("skipping Mail Container in CI environment")
		return
	}

	mailContainerName = os.Getenv("MAIL_CONTAINER_NAME")
	if mailContainerName == "" {
		mailContainerName = "mailcli-mailserver"
	}

	pool, err := common.GetDockerPool()
	if err != nil || pool == nil {
		err = fmt.Errorf("cannot attach to docker: %v", err)
		return
	}

	dh := common.GetDockerHost(pool)
	if dh != "" {
		mailDockerServer = dh
		fmt.Printf("Docker Host: %s\n", mailDockerServer)
	}
	if host := os.Getenv("MAIL_HOST"); host != "" {
		mailDockerServer = host
		fmt.Printf("MAIL_HOST variable is set: %s\n", host)
	}
	fmt.Printf("use mailserver: %s\n", mailDockerServer)

	vendorImagePrefix := os.Getenv("VENDOR_IMAGE_PREFIX")
	repoString := vendorImagePrefix + mailRepo
	fmt.Printf("Try to start docker container for %s:%s\n", repoString, mailRepoTag)

	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: repoString,
		Tag:        mailRepoTag,
		Env: []string{
			"LOG_LEVEL=debug",
			"ONE_DIR=1",
			"POSTFIX_INET_PROTOCOLS=ipv4",
			"PERMIT_DOCKER=connected-networks",
			"ENABLE_OPENDKIM=0",
			"ENABLE_OPENDMARC=0",
			"ENABLE_AMAVIS=0",
			"SSL_TYPE=manual",
			"SSL_CERT_PATH=/tmp/custom-certs/" + mailHostname + "-full.crt",
			"SSL_KEY_PATH=/tmp/custom-certs/" + mailHostname + ".key",
		},
		Hostname: mailHostname,
		Name:     mailContainerName,
		Mounts: []string{
			test.TestDir + "/docker/mail/config:/tmp/docker-mailserver/",
			test.TestDir + "/docker/mail/ssl:/tmp/custom-certs/:ro",
		},
		ExposedPorts: []string{"25", "143", "465", "587", "993"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"25":  {{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", smtpDockerPort)}},
			"143": {{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", imapDockerPort)}},
			"465": {{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", sslDockerPort)}},
			"587": {{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", tlsDockerPort)}},
			"993": {{HostIP: "0.0.0.0", HostPort: fmt.Sprintf("%d", imapsDockerPort)}},
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		err = fmt.Errorf("error starting mailserver docker container: %v", err)
		return
	}

	pool.MaxWait = containerWaitTimeout * time.Second
	fmt.Printf("Wait for mailserver %s:%d (max %ds)...\n", mailDockerServer, tlsDockerPort, containerWaitTimeout)
	start := time.Now()

	var c net.Conn
	if err = pool.Retry(func() error {
		c, err = net.DialTimeout("tcp", net.JoinHostPort(mailDockerServer, fmt.Sprintf("%d", tlsDockerPort)), 2*time.Second)
		return err
	}); err != nil {
		fmt.Printf("Could not connect to Mail Container: %s", err)
		return
	}
	_ = c.Close()

	// allow the mailserver to finish initializing
	time.Sleep(20 * time.Second)
	elapsed := time.Since(start)
	fmt.Printf("Mail Container is available after %s\n", elapsed.Round(time.Millisecond))
	return
}
