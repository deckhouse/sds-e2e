package integration

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

/*  SSH Credentials  */

func (clr *KCluster) GetSSHCredential(name string) (*SSHCredentials, error) {
	sshcredential := &SSHCredentials{}
	err := clr.rtClient.Get(clr.ctx, ctrlrtclient.ObjectKey{Name: name}, sshcredential)
	if err != nil {
		Debugf("Can't get SSHCredential %s: %s", name, err.Error())
		return nil, err
	}
	return sshcredential, nil
}

func (clr *KCluster) ListSSHCredential() ([]SSHCredentials, error) {
	credentials := SSHCredentialsList{}
	err := clr.rtClient.List(clr.ctx, &credentials)
	if err != nil {
		return nil, err
	}

	return credentials.Items, nil
}

func (clr *KCluster) CreateSSHCredential(name, user, privSshKey string) error {
	sshcredential := &SSHCredentials{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: SSHCredentialsSpec{
			User:          user,
			PrivateSSHKey: privSshKey,
		},
	}

	err := clr.rtClient.Create(clr.ctx, sshcredential)
	if err == nil || apierrors.IsAlreadyExists(err) {
		return nil
	}
	Debugf("Can't create SSHCredential %s: %s", name, err.Error())
	return err
}

func (clr *KCluster) CreateOrUpdSSHCredential(name, user, privSshKey string) error {
	if err := clr.CreateSSHCredential(name, user, privSshKey); err != nil {
		return err
	}

	sshcredential, err := clr.GetSSHCredential(name)
	if err != nil {
		return err
	}
	sshcredential.Spec = SSHCredentialsSpec{
		User:          user,
		PrivateSSHKey: privSshKey,
	}
	if err = clr.rtClient.Update(clr.ctx, sshcredential); err != nil {
		Warnf("Can't update SSHCredential %s: %s", name, err.Error())
		return err
	}
	return nil
}

func (clr *KCluster) DeleteSSHCredential(name string) error {
	sshcredential := &SSHCredentials{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return clr.rtClient.Delete(clr.ctx, sshcredential)
}
