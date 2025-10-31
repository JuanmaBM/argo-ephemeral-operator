package argocd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
)

// Client defines the interface for interacting with ArgoCD
type Client interface {
	DoRequestWithRetry(requestFunc func(appClient application.ApplicationServiceClient) error) error
	// CreateApplication creates an ArgoCD Application
	CreateApplication(ctx context.Context, newApp *application.ApplicationCreateRequest) (*v1alpha1.Application, error)
	// GetApplication retrieves an ArgoCD Application
	GetApplication(ctx context.Context, query application.ApplicationQuery) (*v1alpha1.Application, error)
	// GetApplication retrieves an ArgoCD Application
	GetApplications(ctx context.Context) (*v1alpha1.ApplicationList, error)
	// // DeleteApplication deletes an ArgoCD Application
	DeleteApplication(ctx context.Context, name string, namespace string) error
}

// clientImpl implements the Client interface
type clientImpl struct {
	argocdClient apiclient.Client
	tokenLock    sync.Mutex
	username     string
	password     string
	serverAddr   string
	insecure     bool
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func getAuthToken(serverAddr string, username string, password string) (string, error) {
	loginURL := serverAddr + "/api/v1/session"
	loginRequest := LoginRequest{
		Username: username,
		Password: password,
	}

	reqBody, err := json.Marshal(loginRequest)
	if err != nil {
		return "", err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Post(loginURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to login: %s", string(bodyBytes))
	}

	var loginResponse LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		return "", err
	}

	return loginResponse.Token, nil
}

func createArgcdClient(serverAddr string, authToken string, insecure bool) (apiclient.Client, error) {

	clientOpts := &apiclient.ClientOptions{
		ServerAddr: serverAddr,
		AuthToken:  authToken,
		Insecure:   insecure,
		GRPCWeb:    false,
		PlainText:  false,
	}

	client, err := apiclient.NewClient(clientOpts)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewClient(serverAddr string, port string, username string, password string, insecure bool) (Client, error) {

	authToken, err := getAuthToken("https://"+serverAddr, username, password)
	if err != nil {
		log.Fatalf("Client can't get Authorization Token from ArgoCD with the crendetials provided")
		return nil, err
	}

	client, err := createArgcdClient(serverAddr+":"+port, authToken, insecure)
	if err != nil {
		log.Fatalf("Failed to create ArgoCD client: %v", err)
		return nil, err
	}

	return &clientImpl{
		argocdClient: client,
		username:     username,
		password:     password,
		serverAddr:   serverAddr,
		insecure:     insecure,
	}, nil
}

func (c *clientImpl) DoRequestWithRetry(requestFunc func(appClient application.ApplicationServiceClient) error) error {

	conn, appClient, err := c.argocdClient.NewApplicationClient()
	if err != nil {
		return fmt.Errorf("failed to open a connection to ArgoCD server: %v", err)
	}
	defer conn.Close()

	err = requestFunc(appClient)

	if err != nil && isUnauthorized(err) {

		c.tokenLock.Lock()
		defer c.tokenLock.Unlock()

		authToken, err := getAuthToken(c.serverAddr, c.username, c.password)
		if err != nil {
			return fmt.Errorf("error renewing auth token: %v", err)
		}

		c.argocdClient, err = createArgcdClient(c.serverAddr, authToken, c.insecure)
		if err != nil {
			return fmt.Errorf("error recreating ArgoCD client with new token: %v", err)
		}

		err = requestFunc(appClient)
	}

	return err
}

func (c *clientImpl) GetApplications(ctx context.Context) (*v1alpha1.ApplicationList, error) {

	var apps *v1alpha1.ApplicationList
	err := c.DoRequestWithRetry(func(appClient application.ApplicationServiceClient) error {
		appList, err := appClient.List(ctx, &application.ApplicationQuery{})
		if err != nil {
			return fmt.Errorf("failed to get all applications: %v", err)
		}
		apps = appList
		return err
	})

	return apps, err

}

func (c *clientImpl) CreateApplication(ctx context.Context, newApp *application.ApplicationCreateRequest) (*v1alpha1.Application, error) {

	if newApp == nil {
		return nil, errors.New("application must be defined")
	}

	var applicationCreated *v1alpha1.Application
	err := c.DoRequestWithRetry(func(appClient application.ApplicationServiceClient) error {
		app, err := appClient.Create(ctx, newApp)
		if err != nil {
			return fmt.Errorf("application can not be created: %v", err)
		}
		applicationCreated = app
		return err
	})

	return applicationCreated, err
}

func (c *clientImpl) GetApplication(ctx context.Context, query application.ApplicationQuery) (*v1alpha1.Application, error) {

	if isEmpty(query) {
		return nil, errors.New("application name parameter must be defined")
	}

	var foundApp *v1alpha1.Application
	err := c.DoRequestWithRetry(func(appClient application.ApplicationServiceClient) error {
		app, err := appClient.Get(ctx, &query)
		if err != nil {
			log.Fatalf("Application not found with query: %v", query)
		}
		foundApp = app
		return nil
	})

	return foundApp, err
}

func (c *clientImpl) DeleteApplication(ctx context.Context, name string, namespace string) error {

	if name == "" || namespace == "" {
		return errors.New("application name and namespace must be defined")
	}

	return c.DoRequestWithRetry(func(appClient application.ApplicationServiceClient) error {
		_, err := appClient.Delete(ctx, &application.ApplicationDeleteRequest{
			Name: &name,
			// FIXME: AppNamespace is not working as expected, we should investigate why.
			//AppNamespace: &namespace,
		})
		return err
	})
}

func isEmpty(query application.ApplicationQuery) bool {
	fields := []interface{}{
		query.Name,
		query.AppNamespace,
		query.Refresh,
		query.Repo,
		query.ResourceVersion,
		query.Selector,
		query.Project,
		query.Projects,
	}

	for _, field := range fields {
		if field != nil {
			return false
		}

	}

	return true
}

func isUnauthorized(err error) bool {
	if err != nil && strings.Contains(err.Error(), "401") {
		return true
	}
	return false
}
