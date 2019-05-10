package ssh

import (
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"strings"

	"golang.org/x/crypto/ssh"
)

// @brief:
//     Create ssh session auth
// @note:
//     - public key auth
//     - password auth
//     - ssh-agent auth
//     - pkcs11 auth
func (c *Connect) createSshAuth(server string) (auth []ssh.AuthMethod, err error) {
	conf := c.Conf.Server[server]

	// public key (single)
	if conf.Key != "" {
		authMethod, err := createSshAuthPublicKey(conf.Key, conf.KeyPass)
		if err != nil {
			return auth, err
		}
		auth = append(auth, authMethod)
	}

	// public key (multiple)
	if len(conf.Keys) > 0 {
		for _, key := range conf.Keys {
			keyPathArray := strings.SplitN(key, "::", 2)
			authMethod, err := createSshAuthPublicKey(keyPathArray[0], keyPathArray[1])
			if err != nil {
				return auth, err
			}

			auth = append(auth, authMethod)
		}
	}

	// ssh password (single)
	if conf.Pass != "" {
		auth = append(auth, ssh.Password(conf.Pass))
	}

	// ssh password (multiple)
	if len(conf.Passes) > 0 {
		for _, pass := range conf.Passes {
			auth = append(auth, ssh.Password(pass))
		}
	}

	// ssh agent
	if conf.AgentAuth {
		var signers []ssh.Signer
		_, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
		if err != nil {
			signers, err = c.sshAgent.Signers()
			if err != nil {
				return auth, err
			}
		} else {
			signers, err = c.sshExtendedAgent.Signers()

			if err != nil {
				return auth, err
			}
		}
		auth = append(auth, ssh.PublicKeys(signers...))
	}

	if conf.PKCS11Use {
		// @TODO: confのチェック時にPKCS11のProviderのPATHチェックを行う
		var signers []ssh.Signer
		signers, err := c.getSshSignerFromPkcs11(server)
		if err != nil {
			return auth, err
		}

		for _, signer := range signers {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	}

	return auth, err
}

// Craete ssh auth (public key)
func createSshAuthPublicKey(key string, pass string) (auth ssh.AuthMethod, err error) {
	usr, _ := user.Current()
	key = strings.Replace(key, "~", usr.HomeDir, 1)

	// Read PrivateKey file
	keyData, err := ioutil.ReadFile(key)
	if err != nil {
		return auth, err
	}

	// Read signer from PrivateKey
	var signer ssh.Signer
	if pass != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(pass))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}

	// check err
	if err != nil {
		return auth, err
	}

	auth = ssh.PublicKeys(signer)
	return auth, err
}
