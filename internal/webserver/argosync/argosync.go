package argosync

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/argoproj/argo-cd/v3/pkg/apiclient"
	"github.com/argoproj/argo-cd/v3/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v3/pkg/apiclient/session"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	argocdServerAddr = "argocd-server.argocd"
	argocdUser       = "jettisonproj"

	// ArgoCD tokens are valid for 24 hours. Add a refresh threshold to ensure
	// it is refreshed before expiry
	// https://argo-cd.readthedocs.io/en/stable/operator-manual/security/
	tokenRefreshThresholdSeconds int64 = 12 * 60 * 60
	syncOption                         = "CreateNamespace=true"
)

var (
	setupLog = ctrl.Log.WithName("argosync")
)

type SyncClient struct {
	// Contains the auth cred used to retrieve a token
	argocdCred string

	// ArgoCD client used to get new session token
	argocdClientWithoutAuth apiclient.Client

	argocdAuthToken      string
	lastTokenRefreshTime int64
	mutex                sync.Mutex
}

func NewSyncClient(argocdCred string) (*SyncClient, error) {

	// Create ArgoCD client with no auth for starting sessions
	clientOptsWithoutAuth := &apiclient.ClientOptions{
		ServerAddr: argocdServerAddr,
		PlainText:  true,
	}
	argocdClientWithoutAuth, err := apiclient.NewClient(clientOptsWithoutAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to create ArgoCD client without auth: %s", err)
	}

	return &SyncClient{
		argocdCred:              argocdCred,
		argocdClientWithoutAuth: argocdClientWithoutAuth,
	}, nil
}

func (s *SyncClient) Sync(
	ctx context.Context,
	namespace string,
	name string,
) error {

	authToken, err := s.getAuthToken(ctx)
	if err != nil {
		return err
	}

	// Create client with token auth
	clientOptsWithAuth := &apiclient.ClientOptions{
		ServerAddr: argocdServerAddr,
		PlainText:  true,
		AuthToken:  authToken,
	}
	argocdClient, err := apiclient.NewClient(clientOptsWithAuth)
	if err != nil {
		return fmt.Errorf("failed to create ArgoCD client with auth: %s", err)
	}

	// Create ArgoCD application client
	conn, appClient, err := argocdClient.NewApplicationClient()
	if err != nil {
		return fmt.Errorf("failed to create ArgoCD application client: %s", err)
	}
	defer func(conn io.Closer) {
		err := conn.Close()
		if err != nil {
			setupLog.Error(err, "failed to close ArgoCD application connection")
		}
	}(conn)

	// Sync application
	prune := true
	prunePtr := &prune

	syncRequest := &application.ApplicationSyncRequest{
		Prune:        prunePtr,
		Name:         &name,
		AppNamespace: &namespace,
		SyncOptions: &application.SyncOptions{
			Items: []string{syncOption},
		},
	}

	// Call the Sync method on the application client
	_, err = appClient.Sync(ctx, syncRequest)
	if err != nil {
		return fmt.Errorf("failed to sync application %s: %s", name, err)
	}

	return nil
}

func (s *SyncClient) getAuthToken(ctx context.Context) (string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now().Unix()
	if now < s.lastTokenRefreshTime+tokenRefreshThresholdSeconds {
		return s.argocdAuthToken, nil
	}

	log.FromContext(ctx).Info("Refreshing token")

	// Create ArgoCD session client
	conn, sessionClient, err := s.argocdClientWithoutAuth.NewSessionClient()
	if err != nil {
		return "", fmt.Errorf("failed to create ArgoCD session client: %s", err)
	}
	defer func(conn io.Closer) {
		err := conn.Close()
		if err != nil {
			setupLog.Error(err, "failed to close ArgoCD session connection")
		}
	}(conn)

	// Create ArgoCD session
	sessionCreateRequest := &session.SessionCreateRequest{
		Username: argocdUser,
		Password: s.argocdCred,
	}
	sessionResp, err := sessionClient.Create(ctx, sessionCreateRequest)
	if err != nil {
		return "", fmt.Errorf("failed to create ArgoCD session: %s", err)
	}

	s.argocdAuthToken = sessionResp.Token
	s.lastTokenRefreshTime = now
	return s.argocdAuthToken, nil
}
