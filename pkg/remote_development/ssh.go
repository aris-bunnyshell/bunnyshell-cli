package remote_development

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	bunnyshellSSH "bunnyshell.com/cli/pkg/ssh"
	"bunnyshell.com/cli/pkg/util"

	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
)

const (
	PrivateKeyFilename = "id_rsa"
	PublicKeyFilename  = "id_rsa.pub"

	paramForwardAgent          = "ForwardAgent"
	paramHostName              = "HostName"
	paramPort                  = "Port"
	paramStrictHostKeyChecking = "StrictHostKeyChecking"
	paramUserKnownHostsFile    = "UserKnownHostsFile"
	paramIdentityFile          = "IdentityFile"
	paramIdentitiesOnly        = "IdentitiesOnly"

	SyncthingRemoteInterface = "127.0.0.1"
	SyncthingRemotePort      = 22000
)

func (r *RemoteDevelopment) EnsureSSHKeys() error {
	workspace, err := util.GetWorkspaceDir()
	if err != nil {
		return err
	}

	privatePemPath := filepath.Join(workspace, PrivateKeyFilename)
	sshPublicKeyPath := filepath.Join(workspace, PublicKeyFilename)
	_, err1 := os.Stat(privatePemPath)
	_, err2 := os.Stat(sshPublicKeyPath)
	if err1 == nil && err2 == nil {
		r.WithSSH(privatePemPath, sshPublicKeyPath)
		return nil
	}

	spinner := util.MakeSpinner(" Generate SSH RSA key...")
	spinner.Start()
	defer spinner.Stop()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// dump private key
	var privateKeyBytes []byte = x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	privatePemFile, err := os.Create(privatePemPath)
	if err != nil {
		return err
	}
	defer privatePemFile.Close()

	if err := os.Chmod(privatePemPath, 0600); err != nil {
		return err
	}

	err = pem.Encode(privatePemFile, privateKeyBlock)
	if err != nil {
		return err
	}

	// dump public key
	sshPublicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}
	sshPublicKeyBytes := ssh.MarshalAuthorizedKey(sshPublicKey)
	err = os.WriteFile(sshPublicKeyPath, sshPublicKeyBytes, 0600)
	if err != nil {
		return err
	}

	r.WithSSH(privatePemPath, sshPublicKeyPath)
	return err
}

func (r *RemoteDevelopment) EnsureSSHConfigEntry() error {
	config, err := bunnyshellSSH.GetConfig()
	if err != nil {
		return err
	}

	hostname := fmt.Sprintf("%s.bunnyshell", r.ComponentName)
	bunnyshellSSH.RemoveHost(config, hostname)
	host, err := newSSHConfigHost(
		hostname,
		r.KubernetesClient.SSHPortForwardOptions.Interface,
		strconv.Itoa(r.KubernetesClient.SSHPortForwardOptions.LocalPort),
		r.SSHPrivateKeyPath,
	)
	if err != nil {
		return err
	}

	config.Hosts = append(config.Hosts, host)

	return bunnyshellSSH.SaveConfig(config)
}

func newSSHConfigHost(hostname, iface, port, identityFile string) (*ssh_config.Host, error) {
	pattern, err := ssh_config.NewPattern(hostname)
	if err != nil {
		return nil, err
	}
	patterns := []*ssh_config.Pattern{pattern}
	nodes := []ssh_config.Node{
		bunnyshellSSH.NewKV(paramForwardAgent, "yes"),
		bunnyshellSSH.NewKV(paramHostName, iface),
		bunnyshellSSH.NewKV(paramPort, port),
		bunnyshellSSH.NewKV(paramStrictHostKeyChecking, "no"),
		bunnyshellSSH.NewKV(paramUserKnownHostsFile, "/dev/null"),
		bunnyshellSSH.NewKV(paramIdentityFile, identityFile),
		bunnyshellSSH.NewKV(paramIdentitiesOnly, "yes"),
	}
	host := &ssh_config.Host{
		Patterns:   patterns,
		EOLComment: " do not edit - generated by bunnyshell",
		Nodes:      nodes,
	}

	return host, nil
}

func (r *RemoteDevelopment) StartSSHTerminal() error {
	terminal := bunnyshellSSH.NewSSHTerminal(
		r.KubernetesClient.SSHPortForwardOptions.Interface,
		r.KubernetesClient.SSHPortForwardOptions.LocalPort,
		bunnyshellSSH.PrivateKeyFile(r.SSHPrivateKeyPath),
	)

	errChan := make(chan error, 1)
	go func() {
		errChan <- terminal.Start()
		close(errChan)
		r.Close()
	}()

	select {
	case <-terminal.ReadyChannel:
	case err := <-errChan:
		return err
	}

	return nil
}
