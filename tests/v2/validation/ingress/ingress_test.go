package ingresses

import (
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IngressTestSuite struct {
	suite.Suite
	client  *rancher.Client
	session *session.Session
	project *management.Project
}

func (i *IngressTestSuite) TearDownSuite() {
	i.session.Cleanup()
}

func (i *IngressTestSuite) SetupSuite() {
	testSession := session.NewSession()
	i.session = testSession

	client, err := rancher.NewClient("", testSession)
	require.NoError(i.T(), err)

	i.client = client
}
func (i *IngressTestSuite) TestIngress() {
	subSession := i.session.NewSession()
	defer subSession.Cleanup()

	client, err := i.client.WithSession(subSession)
	require.NoError(i.T(), err)

	i.T().Log("creating Ingress")
	//err = //CreateIngress
}
