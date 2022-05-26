package deploy

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/astronomer/astro-cli/airflow"
	"github.com/astronomer/astro-cli/airflow/mocks"
	airflowversions "github.com/astronomer/astro-cli/airflow_versions"
	"github.com/astronomer/astro-cli/astro-client"
	astro_mocks "github.com/astronomer/astro-cli/astro-client/mocks"
	"github.com/astronomer/astro-cli/config"
	"github.com/astronomer/astro-cli/pkg/httputil"
	testUtil "github.com/astronomer/astro-cli/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var errMock = errors.New("mock error")

func TestDeploySuccess(t *testing.T) {
	mockDeplyResp := []astro.Deployment{
		{
			ID:             "test-id",
			ReleaseName:    "test-name",
			RuntimeRelease: astro.RuntimeRelease{Version: "4.2.5"},
			DeploymentSpec: astro.DeploymentSpec{
				Webserver: astro.Webserver{URL: "test-url"},
			},
			CreatedAt: time.Now(),
		},
		{
			ID:             "test-id-2",
			ReleaseName:    "test-name-2",
			RuntimeRelease: astro.RuntimeRelease{Version: "4.2.5"},
			DeploymentSpec: astro.DeploymentSpec{
				Webserver: astro.Webserver{URL: "test-url"},
			},
			CreatedAt: time.Now(),
		},
	}
	testUtil.InitTestConfig(testUtil.CloudPlatform)
	config.CFG.ShowWarnings.SetHomeString("false")
	mockClient := new(astro_mocks.Client)
	mockClient.On("ListWorkspaces").Return([]astro.Workspace{{ID: "test-ws-id"}}, nil).Once()
	mockClient.On("ListDeployments", mock.Anything).Return(mockDeplyResp, nil).Twice()
	mockClient.On("ListPublicRuntimeReleases").Return([]astro.RuntimeRelease{{Version: "4.2.5", AirflowVersion: "2.2.5"}}, nil).Twice()
	mockClient.On("CreateImage", mock.Anything).Return(&astro.Image{}, nil).Twice()
	mockClient.On("DeployImage", mock.Anything).Return(&astro.Image{}, nil).Twice()

	mockImageHandler := new(mocks.ImageHandler)
	airflowImageHandler = func(image string) airflow.ImageHandler {
		mockImageHandler.On("Build", mock.Anything).Return(nil)
		mockImageHandler.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockImageHandler.On("GetLabel", runtimeImageLabel).Return("", nil)
		return mockImageHandler
	}

	mockContainerHandler := new(mocks.ContainerHandler)
	containerHandlerInit = func(airflowHome, envFile, dockerfile string, isPyTestCompose bool) (airflow.ContainerHandler, error) {
		mockContainerHandler.On("Parse", mock.Anything).Return(nil)
		mockContainerHandler.On("Pytest", mock.Anything, mock.Anything).Return("", nil)
		return mockContainerHandler, nil
	}

	ctx, err := config.GetCurrentContext()
	assert.NoError(t, err)
	ctx.Token = "test testing"
	err = ctx.SetContext()
	assert.NoError(t, err)

	// mock os.Stdin
	input := []byte("1")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write(input)
	if err != nil {
		t.Error(err)
	}
	w.Close()
	stdin := os.Stdin
	// Restore stdin right after the test.
	defer func() { os.Stdin = stdin }()
	os.Stdin = r

	err = Deploy("./testfiles/", "", "test-ws-id", "parse", "", true, mockClient)
	assert.NoError(t, err)

	err = Deploy("./testfiles/", "test-id", "test-ws-id", "pytest", "", false, mockClient)
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
	mockImageHandler.AssertExpectations(t)
	mockContainerHandler.AssertExpectations(t)
}

func TestDeployFailure(t *testing.T) {
	// no context set failure
	testUtil.InitTestConfig(testUtil.CloudPlatform)
	err := config.ResetCurrentContext()
	assert.NoError(t, err)
	err = Deploy("./testfiles/", "test-id", "test-ws-id", "pytest", "", true, nil)
	assert.EqualError(t, err, "no context set, have you authenticated to Astro or Astronomer Software? Run astro login and try again")

	// no workspace id provided
	testUtil.InitTestConfig(testUtil.CloudPlatform)
	err = Deploy("./testfiles/", "", "", "parse", "", true, nil)
	assert.EqualError(t, err, "no workspace id provided")

	// airflow parse failure
	mockDeplyResp := []astro.Deployment{
		{
			ID:             "test-id",
			ReleaseName:    "test-name",
			RuntimeRelease: astro.RuntimeRelease{Version: "4.2.5"},
			DeploymentSpec: astro.DeploymentSpec{
				Webserver: astro.Webserver{URL: "test-url"},
			},
		},
	}
	testUtil.InitTestConfig(testUtil.CloudPlatform)
	mockClient := new(astro_mocks.Client)
	mockClient.On("ListWorkspaces").Return([]astro.Workspace{{ID: "test-ws-id"}}, nil).Once()
	mockClient.On("ListDeployments", mock.Anything).Return(mockDeplyResp, nil).Once()
	mockClient.On("ListPublicRuntimeReleases").Return([]astro.RuntimeRelease{{Version: "4.2.5", AirflowVersion: "2.2.5"}}, nil).Once()

	mockImageHandler := new(mocks.ImageHandler)
	airflowImageHandler = func(image string) airflow.ImageHandler {
		mockImageHandler.On("Build", mock.Anything).Return(nil)
		mockImageHandler.On("GetLabel", runtimeImageLabel).Return("4.2.5", nil)
		return mockImageHandler
	}

	mockContainerHandler := new(mocks.ContainerHandler)
	containerHandlerInit = func(airflowHome, envFile, dockerfile string, isPyTestCompose bool) (airflow.ContainerHandler, error) {
		mockContainerHandler.On("Parse", mock.Anything).Return(errMock)
		return mockContainerHandler, nil
	}

	// mock os.Stdin
	input := []byte("1")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write(input)
	if err != nil {
		t.Error(err)
	}
	w.Close()
	stdin := os.Stdin
	// Restore stdin right after the test.
	defer func() { os.Stdin = stdin }()
	os.Stdin = r

	err = Deploy("./testfiles/", "", "test-ws-id", "parse", "", true, mockClient)
	assert.ErrorIs(t, err, errDagsParseFailed)

	mockClient.AssertExpectations(t)
	mockImageHandler.AssertExpectations(t)
	mockContainerHandler.AssertExpectations(t)
}

func TestBuildImageFailure(t *testing.T) {
	testUtil.InitTestConfig(testUtil.CloudPlatform)
	ctx, err := config.GetCurrentContext()
	assert.NoError(t, err)
	ctx.SetSystemAdmin(true)

	mockImageHandler := new(mocks.ImageHandler)
	airflowImageHandler = func(image string) airflow.ImageHandler {
		mockImageHandler.On("Build", mock.Anything).Return(nil)
		mockImageHandler.On("GetLabel", runtimeImageLabel).Return("4.2.5", nil)
		return mockImageHandler
	}

	// dockerfile parsing error
	dockerfile = "Dockerfile.invalid"
	_, err = buildImage(&ctx, "./testfiles/", "4.2.5", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse dockerfile")

	// failed to get runtime releases
	dockerfile = "Dockerfile"
	mockClient := new(astro_mocks.Client)
	mockClient.On("ListInternalRuntimeReleases").Return([]astro.RuntimeRelease{}, errMock).Once()
	_, err = buildImage(&ctx, "./testfiles/", "4.2.5", "", mockClient)
	assert.ErrorIs(t, err, errMock)
	mockClient.AssertExpectations(t)
	mockImageHandler.AssertExpectations(t)
}

func TestIsValidUpgrade(t *testing.T) {
	resp := IsValidUpgrade("4.2.5", "4.2.6")
	assert.True(t, resp)

	resp = IsValidUpgrade("4.2.6", "4.2.5")
	assert.False(t, resp)

	resp = IsValidUpgrade("", "4.2.6")
	assert.True(t, resp)
}

func TestIsValidTag(t *testing.T) {
	resp := IsValidTag([]string{"4.2.5", "4.2.6"}, "4.2.7")
	assert.False(t, resp)

	resp = IsValidTag([]string{"4.2.5", "4.2.6"}, "4.2.6")
	assert.True(t, resp)
}

func TestCheckVersion(t *testing.T) {
	httpClient := airflowversions.NewClient(httputil.NewHTTPClient(), false)
	latestRuntimeVersion, _ := airflowversions.GetDefaultImageTag(httpClient, "")

	// version that is older than newest
	buf := new(bytes.Buffer)
	CheckVersion("1.0.0", buf)
	assert.Contains(t, buf.String(), "WARNING! You are currently running Astro Runtime Version")

	// version that is latest
	CheckVersion(latestRuntimeVersion, buf)
	assert.Contains(t, buf.String(), "Runtime Version: "+latestRuntimeVersion)
}

func TestCheckVersionBeta(t *testing.T) {
	// version that newer than latest
	buf := new(bytes.Buffer)
	defer testUtil.MockUserInput(t, "y")()
	CheckVersion("10.0.0", buf)
	assert.Contains(t, buf.String(), "")
}

func TestPromptUserForDeployment(t *testing.T) {
	_, _, _, _, err := promptUserForDeployment("", &astro.Workspace{}, []astro.Deployment{}) //nolint:dogsled
	assert.ErrorIs(t, err, errNoDeploymentsMsg)

	_, _, _, _, err = promptUserForDeployment("astrodev.io", &astro.Workspace{}, []astro.Deployment{{ID: "test-id-1"}, {ID: "test-id-2"}}) //nolint:dogsled
	assert.ErrorIs(t, err, errInvalidDeploymentKey)
}

func TestCheckPyTest(t *testing.T) {
	mockDeployImage := "test-image"

	mockContainerHandler := new(mocks.ContainerHandler)
	mockContainerHandler.On("Pytest", "", mockDeployImage).Return("", errMock).Once()

	// random error on running airflow pytest
	err := checkPytest("", mockDeployImage, mockContainerHandler)
	assert.ErrorIs(t, err, errMock)
	mockContainerHandler.AssertExpectations(t)

	// airflow pytest exited with status code 1
	mockContainerHandler.On("Pytest", "", mockDeployImage).Return("exit code 1", errMock).Once()
	err = checkPytest("", mockDeployImage, mockContainerHandler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pytests failed, please fix failures or rerun the command without the '--pytest' flag to deploy")
	mockContainerHandler.AssertExpectations(t)
}

func TestValidateWorkspace(t *testing.T) {
	mockClient := new(astro_mocks.Client)
	mockClient.On("ListWorkspaces").Return([]astro.Workspace{}, errMock).Once()

	_, err := validateWorkspace("test-ws-id", mockClient)
	assert.ErrorIs(t, err, errMock)

	mockClient.On("ListWorkspaces").Return([]astro.Workspace{{ID: "test-ws-id"}}, nil).Once()
	_, err = validateWorkspace("test-invalid-id", mockClient)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workspaces with id")

	mockClient.AssertExpectations(t)
}