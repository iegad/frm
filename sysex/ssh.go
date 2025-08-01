package sysex

import (
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func DialSsh(host, user, password string) (*ssh.Client, error) {
	sc, err := ssh.Dial("tcp", host, &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(
				func(user, instruction string, questions []string, echos []bool) ([]string, error) {
					answers := make([]string, len(questions))
					for i := range questions {
						answers[i] = password
					}
					return answers, nil
				},
			),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, err
	}

	return sc, nil
}

func DialSFTP(host, user, password string) (*ssh.Client, *sftp.Client, error) {
	sc, err := DialSsh(host, user, password)
	if err != nil {
		return nil, nil, err
	}

	fc, err := sftp.NewClient(sc)
	if err != nil {
		return nil, nil, err
	}

	return sc, fc, nil
}
