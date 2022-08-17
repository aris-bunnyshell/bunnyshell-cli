package remote_development

import (
	"fmt"
	"net/http"
	"os"

	"bunnyshell.com/cli/pkg/lib"
	"bunnyshell.com/cli/pkg/util"

	bunnysdk "bunnyshell.com/sdk"
)

var (
	ErrNoEnvironments  = fmt.Errorf("no environments")
	ErrNoOrganizations = fmt.Errorf("no organizations")
	ErrNoComponents    = fmt.Errorf("no components")
	ErrNoProjects      = fmt.Errorf("no projects")
)

func (r *RemoteDevelopment) SelectOrganization(defaultOrganizationId string) error {
	if defaultOrganizationId != "" {
		r.OrganizationId = defaultOrganizationId
		return nil
	}

	resp, _, err := getOrganizations()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error when calling `OrganizationApi.OrganizationList`:", err)
		return err
	}

	if resp.Embedded == nil {
		fmt.Fprintln(os.Stderr, "No Organizations")
		return ErrNoOrganizations
	}

	items := []string{}
	for _, item := range resp.Embedded.GetItem() {
		items = append(items, item.GetName())
	}
	index, _, err := util.Choose("Select organization", items)
	if err != nil {
		return err
	}

	r.OrganizationId = resp.Embedded.GetItem()[index].GetId()
	return nil
}

func (r *RemoteDevelopment) SelectProject(defaultProjectId string) error {
	if defaultProjectId != "" {
		r.ProjectId = defaultProjectId
		return nil
	}

	resp, _, err := getProjects(r.OrganizationId)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error when calling `ProjectApi.ProjectList`:", err)
		return err
	}

	if resp.Embedded == nil {
		fmt.Fprintln(os.Stderr, "No Projects")
		return ErrNoProjects
	}

	items := []string{}
	for _, item := range resp.Embedded.GetItem() {
		items = append(items, item.GetName())
	}
	index, _, err := util.Choose("Select project", items)
	if err != nil {
		return err
	}

	r.ProjectId = resp.Embedded.GetItem()[index].GetId()
	return nil
}

func (r *RemoteDevelopment) SelectEnvironment(defaultEnvironmentId string) error {
	if defaultEnvironmentId != "" {
		r.EnvironmentId = defaultEnvironmentId
		return nil
	}

	resp, _, err := getEnvironments(r.OrganizationId)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error when calling `EnvironmentApi.EnvironmentList`:", err)
		return err
	}

	if resp.Embedded == nil {
		fmt.Fprintln(os.Stderr, "No Environments")
		return ErrNoEnvironments
	}

	items := []string{}
	for _, item := range resp.Embedded.GetItem() {
		items = append(items, item.GetName())
	}
	index, _, err := util.Choose("Select environment", items)
	if err != nil {
		return err
	}

	r.EnvironmentId = resp.Embedded.GetItem()[index].GetId()
	return nil
}

func (r *RemoteDevelopment) SelectComponent(defaultComponentId string) error {
	if defaultComponentId != "" {
		component, _, err := getServiceComponent(defaultComponentId)
		if err != nil {
			return err
		}

		r.WithComponent(component)
		return nil
	}

	resp, _, err := getComponents(r.EnvironmentId)

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error when calling `ComponentApi.ComponentList`:", err)
		return err
	}

	if resp.Embedded == nil {
		fmt.Fprintln(os.Stderr, "No Components")
		return ErrNoComponents
	}

	items := []string{}
	for _, item := range resp.Embedded.GetItem() {
		items = append(items, item.GetName())
	}
	index, _, err := util.Choose("Select component", items)
	if err != nil {
		return err
	}

	component := &resp.Embedded.GetItem()[index]
	r.WithComponent(component)
	return nil
}

func (r *RemoteDevelopment) SelectContainer() error {
	podContainers, err := r.KubernetesClient.GetDeploymentContainers(r.ComponentName)
	if err != nil {
		return err
	}

	if len(podContainers) == 1 {
		r.ContainerName = podContainers[0].Name
		return nil
	}

	items := []string{}
	for _, item := range podContainers {
		items = append(items, item.Name)
	}

	index, _, err := util.Choose("Select container", items)
	if err != nil {
		return err
	}

	r.ContainerName = podContainers[index].Name
	return nil
}

func (r *RemoteDevelopment) SelectLocalSyncFolder(defaultLocalSyncPath string) error {
	if defaultLocalSyncPath != "" {
		r.LocalSyncPath = defaultLocalSyncPath
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	localSyncPath, err := util.AskWithDefault("Sync folder", cwd)
	if err != nil {
		return err
	}

	if localSyncPath == "" {
		localSyncPath = cwd
	}

	r.LocalSyncPath = localSyncPath
	return nil
}

func getServiceComponent(componentId string) (*bunnysdk.ComponentItem, *http.Response, error) {
	ctx, cancel := lib.GetContext()
	defer cancel()

	request := lib.GetAPI().ComponentApi.ComponentView(ctx, componentId)

	return request.Execute()
}

func getOrganizations() (*bunnysdk.PaginatedOrganizationCollection, *http.Response, error) {
	ctx, cancel := lib.GetContext()
	defer cancel()

	request := lib.GetAPI().OrganizationApi.OrganizationList(ctx)

	return request.Execute()
}

func getProjects(organization string) (*bunnysdk.PaginatedProjectCollection, *http.Response, error) {
	ctx, cancel := lib.GetContext()
	defer cancel()

	request := lib.GetAPI().ProjectApi.ProjectList(ctx)
	if organization != "" {
		request = request.Organization(organization)
	}

	return request.Execute()
}

func getEnvironments(organization string) (*bunnysdk.PaginatedEnvironmentCollection, *http.Response, error) {
	ctx, cancel := lib.GetContext()
	defer cancel()

	request := lib.GetAPI().EnvironmentApi.EnvironmentList(ctx)
	if organization != "" {
		request = request.Organization(organization)
	}

	return request.Execute()
}

func getComponents(environment string) (*bunnysdk.PaginatedComponentCollection, *http.Response, error) {
	ctx, cancel := lib.GetContext()
	defer cancel()

	request := lib.GetAPI().ComponentApi.ComponentList(ctx)
	if environment != "" {
		request = request.Environment(environment)
	}

	request = request.OperationStatus("running")

	return request.Execute()
}
